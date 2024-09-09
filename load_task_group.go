package main

import (
	"fmt"
	"strings"
)

type LoadTaskGroup struct {
	*BaseTaskGroup
	skuQueryStrings []string
	kwdQueryStrings []string
	loadTasks       []*LoadTask
}

func NewLoadTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, skuQueryStrings []string, kwdQueryStrings []string, normalTasks []*LoadTask) (*LoadTaskGroup, error) {
	loadTaskGroup := &LoadTaskGroup{
		skuQueryStrings: skuQueryStrings,
		loadTasks:       loadTasks,
	}

	baseTaskGroup, err := NewBaseTaskGroup(proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task group: %v", err)
	}

	loadTaskGroup.BaseTaskGroup = baseTaskGroup

	return loadTaskGroup, nil
}

func (g *LoadTaskGroup) AddSkuQuery(skuStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuStr = strings.TrimSpace(strings.ToUpper(skuStr))

	g.skuQueryStrings = append(g.skuQueryStrings, skuStr)
}

func (g *LoadTaskGroup) RemoveSkuQuery(skuStr string) {
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

func (g *LoadTaskGroup) AddKwdQuery(kwdStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	kwdStr = strings.TrimSpace(strings.ToUpper(kwdStr))

	g.kwdQueryStrings = append(g.kwdQueryStrings, kwdStr)
}

func (g *LoadTaskGroup) RemoveKwdQuery(kwdStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	kwdStr = strings.TrimSpace(strings.ToUpper(kwdStr))

	removeIndex := -1
	for i, query := range g.kwdQueryStrings {
		if query == kwdStr {
			removeIndex = i
			break
		}
	}

	if removeIndex >= 0 {
		g.kwdQueryStrings = append(g.kwdQueryStrings[:removeIndex], g.kwdQueryStrings[removeIndex+1:]...)
	}
}
