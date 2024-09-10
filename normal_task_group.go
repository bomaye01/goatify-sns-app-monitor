package main

import (
	"fmt"
	"strings"
)

type NormalTaskGroup struct {
	*BaseTaskGroup
	skuQueryStrings []string
}

func NewNormalTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, skuQueryStrings []string) (*NormalTaskGroup, error) {
	normalTaskGroup := &NormalTaskGroup{
		skuQueryStrings: skuQueryStrings,
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

func (t *NormalTaskGroup) notifySize(productData *ProductData) {}

func (t *NormalTaskGroup) notifyPrice(productData *ProductData, oldPrice string, newPrice string) {}
