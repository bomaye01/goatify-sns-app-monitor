package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type KwdQuery struct {
	rawQueryStr       string
	inclusiveKeywords []string
	exclusiveKeywords []string
	orKeywordsGroups  [][]string
}

type SkuQuery string

type QueryType int

const (
	SkuQueryType QueryType = iota
	KwdQueryType QueryType = iota
)

type LoadTaskGroup struct {
	*BaseTaskGroup
	normalTaskGroup *NormalTaskGroup
	lastKnownPid    string
	skuQueryStrings []string
	kwdQueryStrings []string
	skuQueries      []SkuQuery
	kwdQueries      []KwdQuery
}

func NewLoadTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, lastKnownPid string, skuQueryStrings []string, kwdQueryStrings []string) (*LoadTaskGroup, error) {
	skuQueries := []SkuQuery{}
	for _, queryStr := range skuQueryStrings {
		queryStr = strings.ToUpper(queryStr)
		queryStr = strings.TrimSpace(queryStr)

		skuQueries = append(skuQueries, SkuQuery(queryStr))
	}

	kwdQueries := []KwdQuery{}
	for _, queryStr := range kwdQueryStrings {
		queryStr = strings.ToLower(queryStr)
		queryStr = strings.TrimSpace(queryStr)

		q := createKeywordQuery(queryStr)

		kwdQueries = append(kwdQueries, q)
	}

	loadTaskGroup := &LoadTaskGroup{
		lastKnownPid:    lastKnownPid,
		skuQueryStrings: skuQueryStrings,
		kwdQueryStrings: kwdQueryStrings,
		skuQueries:      skuQueries,
		kwdQueries:      kwdQueries,
	}

	baseTaskGroup, err := NewBaseTaskGroup(" LOAD ", proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task group: %v", err)
	}

	loadTaskGroup.BaseTaskGroup = baseTaskGroup

	return loadTaskGroup, nil
}

func (g *LoadTaskGroup) LinkToNormalTaskGroup(normalTaskGroup *NormalTaskGroup) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if normalTaskGroup == nil {
		return errors.New("error linking normal task group to load task group: normal task group nil")
	}

	g.normalTaskGroup = normalTaskGroup

	return nil
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

func (g *LoadTaskGroup) handleNewArrivalsResponse(res *NewArrivalsResponse) {
	if res == nil || len(res.Response.ProductNodes) == 0 || g.normalTaskGroup == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	newSKUs := []SkuQuery{}

	for _, productNode := range res.Response.ProductNodes {
		if productNode.Pid == g.lastKnownPid {
			break
		}

		newSKUs = append(newSKUs, SkuQuery(productNode.Sku))
	}

	g.normalTaskGroup.AddLoadSkuQueries(newSKUs)

	g.lastKnownPid = res.Response.ProductNodes[0].Pid
}

func (g *LoadTaskGroup) handleSkuCheckResponse(productData []ProductData) {
	g.mu.Lock()
	defer g.mu.Lock()

	syncRequired := false

	for _, product := range productData {
		productSku := SkuQuery(strings.ToUpper(product.Sku))

		matchedBySku := false
		matchingKwdQueries := []string{}

		if g.isLoadSku(productSku) {
			matchedBySku = true
		} else {
			matchingKwdQueries = g.productMatchingKeywordQueries(product)

			if len(matchingKwdQueries) == 0 {
				continue
			}
		}

		g.notifyLoad(product)

		g.normalTaskGroup.AddSkuQuery(product.Sku)

		stateChanged := g.matchProductStates(product.Sku, matchedBySku, matchingKwdQueries)
		if stateChanged {
			syncRequired = true
		}
	}

	if syncRequired {
		go writeProductStates()
	}
}

func (g *LoadTaskGroup) isLoadSku(skuQuery SkuQuery) bool {
	for _, query := range g.skuQueries {
		if query == skuQuery {
			return true
		}
	}

	return false
}

func (g *LoadTaskGroup) matchProductStates(sku string, matchedBySku bool, matchingKeywordQueries []string) bool {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	stateChanged := false

	for _, query := range productStates.Load.NotifiedProducts {
		if query.Sku == sku {
			if query.MatchedBySku != matchedBySku {
				query.MatchedBySku = matchedBySku

				stateChanged = true
			}

			if !reflect.DeepEqual(query.MatchingKeywordQueries, matchingKeywordQueries) {
				query.MatchingKeywordQueries = matchingKeywordQueries

				stateChanged = true
			}
		}
	}

	return stateChanged
}

func (g *LoadTaskGroup) productMatchingKeywordQueries(product ProductData) []string {
	matchingQueries := []string{}

	for _, kwdQuery := range g.kwdQueries {
		match := true
		// Inclusive keywords check
		for _, inclusiveKwd := range kwdQuery.inclusiveKeywords {
			if !strings.Contains(product.IdentifyerStr, inclusiveKwd) {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		// Exclusive keywords check
		for _, exclusiveKwd := range kwdQuery.exclusiveKeywords {
			if strings.Contains(product.IdentifyerStr, exclusiveKwd) {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		// Or inclusive keywords groups check
		for _, orKwdGroup := range kwdQuery.orKeywordsGroups {
			anyMatch := false

			for _, orKwd := range orKwdGroup {
				if strings.Contains(product.IdentifyerStr, orKwd) {
					anyMatch = true
					break
				}
			}

			if !anyMatch {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		matchingQueries = append(matchingQueries, kwdQuery.rawQueryStr)
	}
	return matchingQueries
}

func (g *LoadTaskGroup) notifyLoad(productData ProductData) {}

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
