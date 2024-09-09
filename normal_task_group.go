package main

import (
	"fmt"
	"reflect"
	"strings"
)

type NormalTaskGroup struct {
	*BaseTaskGroup
	skuQueryStrings []string
	normalTasks     []*NormalTask
}

func NewNormalTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, skuQueryStrings []string, normalTasks []*NormalTask) (*NormalTaskGroup, error) {
	normalTaskGroup := &NormalTaskGroup{
		skuQueryStrings: skuQueryStrings,
		normalTasks:     normalTasks,
	}

	baseTaskGroup, err := NewBaseTaskGroup(proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task group: %v", err)
	}

	normalTaskGroup.BaseTaskGroup = baseTaskGroup

	return normalTaskGroup, nil
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

func (t *NormalTask) getChangesToAvailable(knownAvailableSizes []string) []string {
	sizesChangedToAvailable := []string{}

	for _, currentAvailableSize := range t.availableSizes {
		included := false

		for _, knownAvailableSize := range knownAvailableSizes {
			if knownAvailableSize == currentAvailableSize {
				included = true
				break
			}
		}

		if !included {
			sizesChangedToAvailable = append(sizesChangedToAvailable, currentAvailableSize)
		}
	}

	return sizesChangedToAvailable
}

func (t *NormalTask) matchProductStates(productData *ProductData) bool {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()

	stateChange := false

	notifySize := false
	notifyPrice := false

	newAvailableSizes := []string{}
	oldPrice := ""

	for _, productState := range productStates.Normal.ProductStates {
		if productState.ProductPageUrl == t.productPageUrl {
			if !reflect.DeepEqual(productState.AvailableSizes, t.availableSizes) {
				stateChange = true

				newAvailableSizes = t.getChangesToAvailable(productState.AvailableSizes)
				if len(newAvailableSizes) != 0 {
					notifySize = true
				}

				productState.AvailableSizes = t.availableSizes
			}

			if productState.Price != t.price {
				stateChange = true

				if t.price < productState.Price {
					notifyPrice = true
				}

				oldPrice = productState.Price

				productState.Price = t.price
			}
		}
	}

	// Console print
	if notifySize && notifyPrice {
		t.logger.Green(fmt.Sprintf("%s: New available sizes: %v | Price changed: %s -> %s", strings.Split(t.productPageUrl, "product/")[1], newAvailableSizes, oldPrice, t.price))
	} else if notifyPrice {
		if oldPrice > t.price {
			t.logger.Green(fmt.Sprintf("%s: Price changed: %s -> %s", strings.Split(t.productPageUrl, "product/")[1], oldPrice, t.price))
		} else {
			t.logger.Gray(fmt.Sprintf("%s: Price changed: %s -> %s", strings.Split(t.productPageUrl, "product/")[1], oldPrice, t.price))
		}
	} else if notifySize {
		t.logger.Green(fmt.Sprintf("%s: New available sizes: %v", strings.Split(t.productPageUrl, "product/")[1], newAvailableSizes))
	} else {
		t.logger.Gray(fmt.Sprintf("%s: No changes on product", strings.Split(t.productPageUrl, "product/")[1]))
	}

	// Webhook notify
	if notifySize {
		t.notifySize(productData)
	}
	if notifyPrice {
		if oldPrice > t.price {
			t.notifyPrice(productData, oldPrice, t.price)
		}
	}

	return stateChange
}

func (t *NormalTask) notifySize(productData *ProductData) {
	t.webhookHandler.NotifyRestock(productData, t.availableSizes, t.price)
}

func (t *NormalTask) notifyPrice(productData *ProductData, oldPrice string, newPrice string) {
	t.webhookHandler.NotifyPrice(productData, t.availableSizes, oldPrice, newPrice)
}
