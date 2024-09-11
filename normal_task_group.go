package main

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
)

type NormalTaskGroup struct {
	*BaseTaskGroup
	loadTaskGroup   *LoadTaskGroup
	skuQueryStrings []string
	skuQueries      []SkuQuery
	loadSkuQueries  []SkuQuery
}

func NewNormalTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, skuQueryStrings []string) (*NormalTaskGroup, error) {
	skuQueries := []SkuQuery{}
	for _, queryStr := range skuQueryStrings {
		queryStr = strings.ToUpper(queryStr)
		queryStr = strings.TrimSpace(queryStr)

		skuQueries = append(skuQueries, SkuQuery(queryStr))
	}

	normalTaskGroup := &NormalTaskGroup{
		skuQueryStrings: skuQueryStrings,
		skuQueries:      skuQueries,
		loadSkuQueries:  []SkuQuery{},
	}

	baseTaskGroup, err := NewBaseTaskGroup("NORMAL", proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task group: %v", err)
	}

	normalTaskGroup.BaseTaskGroup = baseTaskGroup

	return normalTaskGroup, nil
}

func (g *NormalTaskGroup) LinkToLoadTaskGroup(loadTaskGroup *LoadTaskGroup) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if loadTaskGroup == nil {
		return errors.New("error linking load task group to normal task group: load task group nil")
	}

	g.loadTaskGroup = loadTaskGroup

	return nil
}

func (g *NormalTaskGroup) AddSkuQuery(skuStr string, productData ProductData) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuStr = strings.TrimSpace(strings.ToUpper(skuStr))

	g.skuQueryStrings = append(g.skuQueryStrings, skuStr)
	g.skuQueries = append(g.skuQueries, SkuQuery(skuStr))

	newState := &ProductStateNormal{
		Sku:            productData.Sku,
		AvailableSizes: productData.AvailableSizes,
		Price:          productData.Price,
	}

	statesNormalMu.Lock()
	NormalSetState(skuStr, newState)
	statesNormalMu.Unlock()

	go writeProductStates()
}

func (g *NormalTaskGroup) RemoveSkuQuery(skuStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuStr = strings.TrimSpace(strings.ToUpper(skuStr))

	removeIndex := -1
	for i, query := range g.skuQueryStrings {
		if query == skuStr {
			removeIndex = i
			break
		}
	}

	if removeIndex == -1 {
		return
	}

	g.skuQueryStrings = append(g.skuQueryStrings[:removeIndex], g.skuQueryStrings[removeIndex+1:]...)

	statesNormalMu.Lock()
	NormalUnsetState(skuStr)
	statesNormalMu.Unlock()

	go writeProductStates()
}

func (g *NormalTaskGroup) checkProductsBySkusResponse(res *ProductsBySkusResponse) {
	if res == nil || len(res.Data.Site.Search.SearchProducts.Products.Edges) == 0 || g.loadTaskGroup == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	syncRequired := false

	checkedLoadQueries := []SkuQuery{}
	loadProductData := []ProductData{}

	includedSkuQueries := make(map[SkuQuery]bool)
	for _, productEdge := range res.Data.Site.Search.SearchProducts.Products.Edges {
		pSkuQuery := MakeSkuQuery(productEdge.Node.Sku)

		includedSkuQueries[pSkuQuery] = true

		// Determine if sku is from normal or load
		productData := GetProductData(productEdge.Node)

		if g.isNormalSku(pSkuQuery) {
			stateChanged := g.matchProductStates(productData)
			if stateChanged {
				syncRequired = true
			}
		} else if g.isLoadSku(pSkuQuery) {
			checkedLoadQueries = append(checkedLoadQueries, pSkuQuery)
			loadProductData = append(loadProductData, productData)
		}

	}

	for _, skuQuery := range g.skuQueries {
		if !includedSkuQueries[skuQuery] {
			syncRequired = true

			statesNormalMu.Lock()
			resetStates := &ProductStateNormal{
				Sku:            string(skuQuery),
				Price:          "0",
				AvailableSizes: []AvailableSize{},
			}

			NormalSetState(resetStates.Sku, resetStates)
			statesNormalMu.Unlock()
		}
	}

	if syncRequired {
		go writeProductStates()
	}

	if len(loadProductData) == 0 {
		return
	}

	go g.loadTaskGroup.handleSkuCheckResponse(loadProductData)

	g.removeCheckedLoadSkuQueries(checkedLoadQueries)
}

func (g *NormalTaskGroup) matchProductStates(product ProductData) bool {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	stateChange := false

	notifySize := false
	notifyPrice := false

	newAvailableSizes := []AvailableSize{}
	oldPrice := ""

	productInStates := false

	for _, state := range productStates.Normal.ProductStates {
		if state.Sku == product.Sku {
			productInStates = true

			if !reflect.DeepEqual(state.AvailableSizes, product.AvailableSizes) {
				stateChange = true

				newAvailableSizes = getChangesToAvailable(state.AvailableSizes, product.AvailableSizes)
				if len(newAvailableSizes) != 0 {
					notifySize = true
				}

				state.AvailableSizes = product.AvailableSizes
			}

			if state.Price != product.Price {
				stateChange = true

				if product.Price < state.Price {
					notifyPrice = true
				}

				oldPrice = state.Price

				state.Price = product.Price
			}
		}
	}

	if !productInStates {
		newState := &ProductStateNormal{
			Sku:            product.Sku,
			AvailableSizes: product.AvailableSizes,
			Price:          product.Price,
		}

		NormalSetState(newState.Sku, newState)

		stateChange = true
	}

	// Console log
	if notifySize && notifyPrice {
		g.logger.Green(fmt.Sprintf("%s: New available sizes: %v | Price changed: %s -> %s", product.Sku, newAvailableSizes, oldPrice, product.Price))
	} else if notifyPrice {
		if oldPrice > product.Price {
			g.logger.Green(fmt.Sprintf("%s: Price changed: %s -> %s", product.Sku, oldPrice, product.Price))
		} else {
			g.logger.Gray(fmt.Sprintf("%s: Price changed: %s -> %s", product.Sku, oldPrice, product.Price))
		}
	} else if notifySize {
		g.logger.Green(fmt.Sprintf("%s: New available sizes: %v", product.Sku, newAvailableSizes))
	} else {
		g.logger.Gray(fmt.Sprintf("%s: No changes on product", product.Sku))
	}

	// Webhook notify
	if notifySize {
		g.notifySize(product)
	}
	if notifyPrice {
		if oldPrice > product.Price {
			g.notifyPrice(product, oldPrice)
		}
	}

	return stateChange
}

func (g *NormalTaskGroup) isNormalSku(sku SkuQuery) bool {
	for _, query := range g.skuQueries {
		if query == sku {
			return true
		}
	}
	return false
}

func (g *NormalTaskGroup) isLoadSku(sku SkuQuery) bool {
	for _, query := range g.loadSkuQueries {
		if query == sku {
			return true
		}
	}
	return false
}

func (g *NormalTaskGroup) getAllSkusAsStrings() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	allSkus := []string{}

	existing := make(map[SkuQuery]bool, len(g.loadSkuQueries)+len(g.skuQueries))

	for _, normalQuery := range g.skuQueries {
		allSkus = append(allSkus, string(normalQuery))

		existing[normalQuery] = true
	}
	for _, loadQuery := range g.loadSkuQueries {
		if !existing[loadQuery] {
			allSkus = append(allSkus, string(loadQuery))

			existing[loadQuery] = true
		}
	}

	return allSkus
}

func (g *NormalTaskGroup) AddLoadSkuQueries(queries []SkuQuery) {
	g.mu.Lock()
	defer g.mu.Unlock()

	existing := make(map[SkuQuery]bool, len(g.loadSkuQueries))

	for _, q := range g.loadSkuQueries {
		existing[q] = true
	}

	for _, q := range queries {
		if !existing[q] {
			g.loadSkuQueries = append(g.loadSkuQueries, q)
			existing[q] = true
		}
	}

	log.Printf("Added %d load sku queries: %v\n", len(queries), queries) // DEBUG
}

func (g *NormalTaskGroup) removeCheckedLoadSkuQueries(checkedQueries []SkuQuery) bool {
	uncheckedQueries := []SkuQuery{}

	for _, query := range g.loadSkuQueries {
		checked := false

		for _, checkedQuery := range checkedQueries {
			if query == checkedQuery {
				checked = true
				break
			}
		}

		if !checked {
			uncheckedQueries = append(uncheckedQueries, query)
		}
	}

	g.loadSkuQueries = uncheckedQueries

	return false
}

func (t *NormalTaskGroup) notifySize(productData ProductData) {
	webhookHandler.NotifyRestock(productData)
}

func (t *NormalTaskGroup) notifyPrice(productData ProductData, oldPrice string) {
	webhookHandler.NotifyPrice(productData, oldPrice)
}
