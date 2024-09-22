package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	SKUS_BATCH_SIZE  = 30
	UNLOAD_THRESHOLD = 100 // 100 requests required to unload product state
)

type NormalTaskGroup struct {
	*BaseTaskGroup
	loadTaskGroup  *LoadTaskGroup
	nextPosToCheck int
	skuQueries     []SkuQuery
	loadSkuQueries []SkuQuery
	unloadCount    map[SkuQuery]int
}

func NewNormalTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, skuQueryStrings []string) (*NormalTaskGroup, error) {
	skuQueries := []SkuQuery{}
	for _, queryStr := range skuQueryStrings {
		queryStr = strings.ToUpper(queryStr)
		queryStr = strings.TrimSpace(queryStr)

		skuQueries = append(skuQueries, SkuQuery(queryStr))
	}

	normalTaskGroup := &NormalTaskGroup{
		skuQueries:     skuQueries,
		loadSkuQueries: []SkuQuery{},
		unloadCount:    make(map[SkuQuery]int),
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

	g.skuQueries = append(g.skuQueries, SkuQuery(skuStr))

	newState := &ProductStateNormal{
		Sku:              skuStr,
		AvailableForSale: true,
		AvailableSizes:   []AvailableSize{},
		Price:            "0",
	}

	statesNormalMu.Lock()
	NormalSetState(skuStr, newState)
	statesNormalMu.Unlock()

	go writeProductStates()
}

func (g *NormalTaskGroup) RemoveSkuQuery(skuStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuQuery := MakeSkuQuery(skuStr)

	removeIndex := -1
	for i, query := range g.skuQueries {
		if query == skuQuery {
			removeIndex = i
			break
		}
	}

	if removeIndex >= 0 {
		g.skuQueries = append(g.skuQueries[:removeIndex], g.skuQueries[removeIndex+1:]...)
	}

	statesNormalMu.Lock()
	NormalUnsetState(string(skuQuery))
	statesNormalMu.Unlock()

	go writeProductStates()
}

func (g *NormalTaskGroup) checkProductsBySkusResponse(res *ProductsBySkusResponse, skusInRequest []string) {
	if res == nil || g.loadTaskGroup == nil {
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
		g.unloadCount[pSkuQuery] = 0

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

	for _, skuQueryStr := range skusInRequest {
		if skuQuery := MakeSkuQuery(skuQueryStr); !includedSkuQueries[skuQuery] {
			if g.unloadCount[skuQuery]+1 == UNLOAD_THRESHOLD {
				g.logger.Grey(fmt.Sprintf("%s: Not loaded. Resetting product state...", string(skuQuery)))

				statesNormalMu.Lock()
				resetStates := &ProductStateNormal{
					Sku:              string(skuQuery),
					AvailableForSale: true,
					Price:            "0",
					AvailableSizes:   []AvailableSize{},
				}

				NormalSetState(resetStates.Sku, resetStates)
				statesNormalMu.Unlock()

				g.unloadCount[skuQuery] = UNLOAD_THRESHOLD + 1 // Make sure to only reset once
			} else {
				g.logger.Grey(fmt.Sprintf("%s: Not loaded", string(skuQuery)))

				g.unloadCount[skuQuery] += 1
			}

			syncRequired = true
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

	isAvailableForSale := product.AvailableForSale

	notifySize := false
	notifyPrice := false
	notifyAvailableForSale := false

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

				notifyPrice = true

				oldPrice = state.Price

				state.Price = product.Price
			}

			if state.AvailableForSale != product.AvailableForSale {
				if !state.AvailableForSale {
					notifyAvailableForSale = true
				}

				state.AvailableForSale = product.AvailableForSale
			}
		}
	}

	if !productInStates {
		newState := &ProductStateNormal{
			Sku:              product.Sku,
			AvailableForSale: true,
			AvailableSizes:   product.AvailableSizes,
			Price:            product.Price,
		}

		NormalSetState(newState.Sku, newState)

		stateChange = true
	}

	// Console log
	if notifySize || notifyPrice || notifyAvailableForSale {
		notifyStr := fmt.Sprintf("%s:", product.Sku)
		colorGreen := false

		if notifySize {
			notifyStr = fmt.Sprintf("%s New available sizes: %v |", notifyStr, newAvailableSizes)
			colorGreen = true
		}
		if notifyPrice {
			notifyStr = fmt.Sprintf("%s Price changed: %s -> %s |", notifyStr, oldPrice, product.Price)
			if oldPrice > product.Price {
				colorGreen = true
			}
		}
		if notifyAvailableForSale {
			notifyStr = fmt.Sprintf("%s available for sale", notifyStr)
			colorGreen = true
		}
		notifyStr = strings.TrimSuffix(notifyStr, " |")

		if !isAvailableForSale {
			colorGreen = false
		}

		if colorGreen {
			g.logger.Green(notifyStr)
		} else {
			g.logger.Grey(notifyStr)
		}
	} else {
		g.logger.Grey(fmt.Sprintf("%s: No changes", product.Sku))
	}

	// Webhook notify
	if notifySize && isAvailableForSale {
		g.notifySize(product)
	}
	if notifyPrice {
		if oldPrice > product.Price {
			g.notifyPrice(product, oldPrice)
		}
	}
	if notifyAvailableForSale {
		g.notifyAvailable(product)
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

func (g *NormalTaskGroup) getNextSkus() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	nextSkus := []string{}

	existing := make(map[SkuQuery]bool, len(g.loadSkuQueries)+len(g.skuQueries))

	// Always add load first to prevent starvation
	for _, loadQuery := range g.loadSkuQueries {
		if !existing[loadQuery] {
			nextSkus = append(nextSkus, string(loadQuery))

			existing[loadQuery] = true
		}

		if len(nextSkus) == SKUS_BATCH_SIZE {
			return nextSkus
		}
	}

	// Then add normal skus
	pointer := g.nextPosToCheck

	for len(nextSkus) < SKUS_BATCH_SIZE && len(g.skuQueries) > 0 {
		// Append if not included already
		normalQuery := g.skuQueries[pointer]
		if !existing[normalQuery] {
			nextSkus = append(nextSkus, string(normalQuery))

			existing[normalQuery] = true
		}

		// Increment
		pointer += 1

		// Round robin
		if pointer == len(g.skuQueries) {
			pointer = 0
		}

		// Break condition
		if pointer == g.nextPosToCheck {
			break
		}
	}

	g.nextPosToCheck = pointer

	return nextSkus
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

func (t *NormalTaskGroup) notifyAvailable(productData ProductData) {
	webhookHandler.NotifyAvailable(productData)
}
