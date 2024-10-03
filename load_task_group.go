package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	LOAD_CHECK_RETRIES = 15
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
	kwdQueries      []KwdQuery
}

func NewLoadTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, lastKnownPid string, kwdQueryStrings []string) (*LoadTaskGroup, error) {
	kwdQueries := []KwdQuery{}
	for _, queryStr := range kwdQueryStrings {
		queryStr = strings.ToLower(queryStr)
		queryStr = strings.TrimSpace(queryStr)

		q := MakeKeywordQuery(queryStr)

		kwdQueries = append(kwdQueries, q)
	}

	loadTaskGroup := &LoadTaskGroup{
		lastKnownPid: lastKnownPid,
		kwdQueries:   kwdQueries,
	}

	baseTaskGroup, err := NewBaseTaskGroup("LOAD", proxyHandler, webhookHandler)
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

func (g *LoadTaskGroup) AddKwdQuery(kwdStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	kwdStr = strings.TrimSpace(strings.ToLower(kwdStr))

	query := MakeKeywordQuery(kwdStr)

	g.kwdQueries = append(g.kwdQueries, query)

	statesLoadMu.Lock()
	LoadAddKwd(query.rawQueryStr)
	statesLoadMu.Unlock()

	go writeProductStates()
}

func (g *LoadTaskGroup) RemoveKwdQuery(kwdStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	kwdStr = strings.TrimSpace(strings.ToLower(kwdStr))

	for strings.Contains(kwdStr, "  ") {
		kwdStr = strings.Replace(kwdStr, "  ", " ", -1)
	}

	removeIndex := -1
	for i, query := range g.kwdQueries {
		if query.rawQueryStr == kwdStr {
			removeIndex = i
			break
		}
	}

	if removeIndex >= 0 {
		g.kwdQueries = append(g.kwdQueries[:removeIndex], g.kwdQueries[removeIndex+1:]...)
	}

	statesLoadMu.Lock()
	LoadRemoveKwd(kwdStr)
	statesLoadMu.Unlock()

	go writeProductStates()
}

func (g *LoadTaskGroup) handleNewArrivalsResponse(res *NewArrivalsResponse) {
	if g.normalTaskGroup == nil || res == nil || len(res.Response.ProductNodes) == 0 {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	newSKUs := []SkuQuery{}

	for _, productNode := range res.Response.ProductNodes {
		if productNode.Pid == g.lastKnownPid {
			break
		}

		newSKUs = append(newSKUs, MakeSkuQuery(productNode.Sku))
	}

	if numNewSkus := len(newSKUs); numNewSkus > 0 {
		g.logger.Yellow(fmt.Sprintf("%d new products loaded.", numNewSkus))

		for len(newSKUs) > SKUS_BATCH_SIZE {
			nextSKUs := newSKUs[:SKUS_BATCH_SIZE]

			go g.loadCheckSkus(nextSKUs)

			newSKUs = newSKUs[SKUS_BATCH_SIZE:]
		}
		if len(newSKUs) > 0 {
			go g.loadCheckSkus(newSKUs)
		}

		g.lastKnownPid = res.Response.ProductNodes[0].Pid

		statesLoadMu.Lock()
		LoadSetLastKnownPid(g.lastKnownPid)
		statesLoadMu.Unlock()

		go writeProductStates()
	} else {
		g.logger.Grey("No new products loaded")
	}
}

func (g *LoadTaskGroup) loadCheckSkus(skus []SkuQuery) {
	var skusStr []string

	loadTask, err := NewLoadTask("Load Check", g)
	if err != nil {
		g.logger.Red(err)
		return
	}

	includedSkuQueries := make(map[SkuQuery]bool)

	configMu.RLock()
	timeout := config.LoadTask.Timeout
	configMu.RUnlock()

	for i := range LOAD_CHECK_RETRIES {
		skusStr = []string{}
		for _, sku := range skus {
			skusStr = append(skusStr, string(sku))
		}

		if len(skusStr) == 0 {
			break
		}

		g.logger.Yellow(fmt.Sprintf("Requesting new products: %d (%d/%d)", len(skusStr), i+1, LOAD_CHECK_RETRIES))

		res, err := loadTask.getProductsBySku(skusStr)
		if err != nil {
			g.logger.Red(err)
			return
		}

		if res == nil || res.Data.Site.Search.SearchProducts.Products.Edges == nil {
			continue
		}

		loadProductData := []ProductData{}

		for _, productEdge := range res.Data.Site.Search.SearchProducts.Products.Edges {
			if productEdge.Node.Variants == nil {
				g.logger.Red(fmt.Sprintf("%s: Variants property nil. Skipping product edge...", productEdge.Node.Sku))
				continue
			}
			if productEdge.Node.Variants.Edges == nil {
				g.logger.Red(fmt.Sprintf("%s: Variants.Edges property nil. Skipping product edge...", productEdge.Node.Sku))
				continue
			}

			pSkuQuery := MakeSkuQuery(productEdge.Node.Sku)
			includedSkuQueries[pSkuQuery] = true

			productData := GetProductData(productEdge.Node)

			loadProductData = append(loadProductData, productData)
		}

		go g.handleSkuCheckResponse(loadProductData)

		uncheckedSkus := []SkuQuery{}
		for _, sku := range skus {
			if !includedSkuQueries[sku] {
				uncheckedSkus = append(uncheckedSkus, sku)
			}
		}

		skus = uncheckedSkus

		time.Sleep(time.Millisecond * time.Duration(timeout))
	}
}

func (g *LoadTaskGroup) handleSkuCheckResponse(productData []ProductData) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.logger.Yellow(fmt.Sprintf("Checking %d new products", len(productData)))

	syncRequired := false

	for _, product := range productData {
		matchingKwdQueries := g.keywordQueriesMatchingProduct(product)

		if len(matchingKwdQueries) == 0 {
			continue
		}

		// Dont ping skus that are already in normal monitor
		if g.normalTaskGroup.isNormalSku(MakeSkuQuery(product.Sku)) {
			continue
		}

		// Dont ping already notified products
		statesLoadMu.Lock()

		alreadyNotified := false
		for _, notifiedState := range productStates.Load.NotifiedProducts {
			if notifiedState.Sku == product.Sku {
				alreadyNotified = true
				break
			}
		}

		statesLoadMu.Unlock()

		if alreadyNotified {
			continue
		}

		// Notify
		g.logger.Green(fmt.Sprintf("%s loaded. Matching keywords: %v", product.Sku, matchingKwdQueries))
		g.notifyLoad(product, matchingKwdQueries)

		stateChanged := g.matchProductStates(product.Sku, matchingKwdQueries)
		if stateChanged {
			syncRequired = true
		}
	}

	if syncRequired {
		go writeProductStates()
	}
}

func (g *LoadTaskGroup) matchProductStates(sku string, matchingKeywordQueries []string) bool {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	stateChanged := false
	included := false

	for _, query := range productStates.Load.NotifiedProducts {
		if query.Sku == sku {
			included = true

			if !reflect.DeepEqual(query.MatchingKeywordQueries, matchingKeywordQueries) {
				query.MatchingKeywordQueries = matchingKeywordQueries

				stateChanged = true
			}
		}
	}

	if !included {
		stateChanged = true

		newNotifiedProduct := &ProductStateLoad{
			Sku:                    sku,
			MatchingKeywordQueries: matchingKeywordQueries,
		}

		productStates.Load.NotifiedProducts = append(productStates.Load.NotifiedProducts, newNotifiedProduct)
	}

	return stateChanged
}

func (g *LoadTaskGroup) keywordQueriesMatchingProduct(product ProductData) []string {
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

func (g *LoadTaskGroup) notifyLoad(productData ProductData, matchingKwdQueries []string) {
	webhookHandler.NotifyLoad(productData, matchingKwdQueries)
}

func MakeSkuQuery(skuStr string) SkuQuery {
	return SkuQuery(strings.TrimSpace(strings.ToUpper(skuStr)))
}

func MakeKeywordQuery(kwdSearchQuery string) KwdQuery {
	kwdSearchQuery = strings.TrimSpace(strings.ToLower(kwdSearchQuery))

	for strings.Contains(kwdSearchQuery, "  ") {
		kwdSearchQuery = strings.Replace(kwdSearchQuery, "  ", " ", -1)
	}

	kwdGroup := KwdQuery{
		rawQueryStr: kwdSearchQuery,
	}

	kwdSplitSequence := fmt.Sprintf(" %s", kwdSearchQuery)

	rawInclusiveKwds := strings.Split(kwdSplitSequence, " +")
	rawExclusiveKwds := strings.Split(kwdSplitSequence, " -")

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
		kwd = strings.ReplaceAll(kwd, "+", " ")
		kwd = strings.ReplaceAll(kwd, "-", " ")

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
		kwd = strings.ReplaceAll(kwd, "+", " ")
		kwd = strings.ReplaceAll(kwd, "-", " ")

		kwdGroup.exclusiveKeywords = append(kwdGroup.exclusiveKeywords, kwd)
	}

	return kwdGroup
}
