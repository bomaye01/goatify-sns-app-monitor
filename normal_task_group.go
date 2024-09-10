package main

import (
	"errors"
	"fmt"
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

func (g *NormalTaskGroup) AddSkuQuery(skuStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuStr = strings.TrimSpace(strings.ToUpper(skuStr))

	g.skuQueryStrings = append(g.skuQueryStrings, skuStr)
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

	if removeIndex >= 0 {
		g.skuQueryStrings = append(g.skuQueryStrings[:removeIndex], g.skuQueryStrings[removeIndex+1:]...)
	}
}

func (g *NormalTaskGroup) checkProductsBySkusResponse(res *ProductsBySkusResponse) {
	if res == nil || len(res.Data.Site.Search.SearchProducts.Products.Edges) == 0 || g.loadTaskGroup == nil {
		return
	}

	g.mu.Lock()

	syncRequired := false

	checkedLoadQueries := []SkuQuery{}
	loadProductData := []ProductData{}

	for _, productEdge := range res.Data.Site.Search.SearchProducts.Products.Edges {
		// Determine if sku is from normal or load
		pSkuQuery := SkuQuery(strings.ToUpper(productEdge.Node.Sku))

		productData := GetProductData(productEdge.Node)

		if g.isNormalSku(pSkuQuery) {
			stateChanged := g.matchProductStates(productData)
			if stateChanged {
				syncRequired = true
			}
		}
		if g.isLoadSku(pSkuQuery) {
			checkedLoadQueries = append(checkedLoadQueries, pSkuQuery)
			loadProductData = append(loadProductData, productData)
		}
	}

	go g.loadTaskGroup.handleSkuCheckResponse(loadProductData)

	g.removeCheckedLoadSkuQueries(checkedLoadQueries)

	g.mu.Unlock()

	if syncRequired {
		go writeProductStates()
	}
}

func (g *NormalTaskGroup) matchProductStates(product ProductData) bool {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()

	stateChange := false

	notifySize := false
	notifyPrice := false

	newAvailableSizes := []string{}
	oldPrice := ""

	for _, state := range productStates.Normal.ProductStates {
		if state.Sku == product.Sku {
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
		g.notifySize(&product)
	}
	if notifyPrice {
		if oldPrice > product.Price {
			g.notifyPrice(&product, oldPrice, product.Price)
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

func (g *NormalTaskGroup) getAllSkusAsStrings() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	allSkus := []string{}

	for _, q := range g.skuQueries {
		allSkus = append(allSkus, string(q))
	}
	for _, q := range g.loadSkuQueries {
		allSkus = append(allSkus, string(q))
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
}

func (t *NormalTaskGroup) notifySize(productData *ProductData) {}

func (t *NormalTaskGroup) notifyPrice(productData *ProductData, oldPrice string, newPrice string) {}
