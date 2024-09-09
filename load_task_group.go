package main

import (
	"strings"
	"sync"
)

type LoadTaskGroup struct {
	mu              sync.Mutex
	skuQueryStrings []string
	kwdQueryStrings []string
	loadTasks       []*LoadTask
}

func NewLoadTaskGroup(skuQueryStrings []string, kwdQueryStrings []string, loadTasks []*LoadTask) *LoadTaskGroup {
	return &LoadTaskGroup{
		skuQueryStrings: skuQueryStrings,
		kwdQueryStrings: kwdQueryStrings,
		loadTasks:       loadTasks,
	}
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
