package main

import (
	"fmt"
	"strings"
)

type LoadTaskGroup struct {
	*BaseTaskGroup
	lastKnownPid    string
	skuQueryStrings []string
	kwdQueryStrings []string
}

func NewLoadTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, lastKnownPid string, skuQueryStrings []string, kwdQueryStrings []string) (*LoadTaskGroup, error) {
	loadTaskGroup := &LoadTaskGroup{
		lastKnownPid:    lastKnownPid,
		skuQueryStrings: skuQueryStrings,
		kwdQueryStrings: kwdQueryStrings,
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

func createKeywordQuery(kwdSearchQuery string) KwdQuery {
	kwdGroup := KwdQuery{
		rawQueryStr: kwdSearchQuery,
	}

	kwdSearchQuery = fmt.Sprintf(" %s", kwdSearchQuery)

	for strings.Contains(kwdSearchQuery, "  ") {
		kwdSearchQuery = strings.Replace(kwdSearchQuery, "  ", " ", -1)
	}

	rawInclusiveKwds := strings.Split(kwdSearchQuery, " +")
	rawExclusiveKwds := strings.Split(kwdSearchQuery, " -")

	for _, kwd := range rawInclusiveKwds {
		kwd = strings.ToLower(kwd)
		if len(kwd) == 0 {
			continue
		}
		if indexDelimiter := strings.Index(kwd, " -"); indexDelimiter >= 0 {
			if indexDelimiter == 0 {
				continue
			}
			kwd = kwd[:indexDelimiter]
		}
		kwd = strings.TrimSpace(kwd)

		if strings.Contains(kwd, "/") {
			orKwdGroup := strings.Split(kwd, "/")

			for i, orKwd := range orKwdGroup {
				orKwdGroup[i] = strings.TrimSpace(orKwd)
			}

			kwdGroup.orKeywordsGroups = append(kwdGroup.orKeywordsGroups, orKwdGroup)
		} else {
			kwdGroup.inclusiveKeywords = append(kwdGroup.inclusiveKeywords, kwd)
		}
	}

	for _, kwd := range rawExclusiveKwds {
		kwd = strings.ToLower(kwd)
		if len(kwd) == 0 {
			continue
		}
		if indexDelimiter := strings.Index(kwd, " +"); indexDelimiter >= 0 {
			if indexDelimiter == 0 {
				continue
			}
			kwd = kwd[:indexDelimiter]
		}
		kwd = strings.TrimSpace(kwd)

		kwdGroup.exclusiveKeywords = append(kwdGroup.exclusiveKeywords, kwd)
	}

	return kwdGroup
}
