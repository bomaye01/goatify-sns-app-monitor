package main

import (
	"strings"
	"sync"
)

type NormalTaskGroup struct {
	mu              sync.Mutex
	skuQueryStrings []string
	normalTasks     []*NormalTask
}

func NewNormalTaskGroup(skuQueryStrings []string, kwdQueryStrings []string, normalTasks []*NormalTask) *NormalTaskGroup {
	return &NormalTaskGroup{
		skuQueryStrings: skuQueryStrings,
		normalTasks:     normalTasks,
	}
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
