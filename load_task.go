package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
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
	SkuQueryType     QueryType = iota
	KeywordQueryType QueryType = iota
)

type LoadTask struct {
	*SnsTask
	skuQueries             []SkuQuery
	kwdQueries             []KwdQuery
	lastKnownPid           string
	numAttemptsLoadProduct int
	syncRequired           bool
}

type productWrapper struct {
	productData      *ProductData
	productsPageBody ProductsPageBody
}

func NewLoadTask(taskName string, proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, strSkuQueries []string, strKwdQueries []string, lastKnownPid string) (*LoadTask, error) {
	skuQueries := []SkuQuery{}
	kwdQueries := []KwdQuery{}

	for _, strSkuQuery := range strSkuQueries {
		strSkuQuery = strings.ToUpper(string(strSkuQuery))
		skuQuery := SkuQuery(strSkuQuery)

		skuQueries = append(skuQueries, SkuQuery(skuQuery))
	}
	for _, strKwdQuery := range strKwdQueries {
		strKwdQuery = strings.ToLower(strKwdQuery)
		kwdquery := createKeywordQuery(strKwdQuery)

		kwdQueries = append(kwdQueries, kwdquery)
	}

	loadTask := &LoadTask{
		SnsTask:                &SnsTask{},
		skuQueries:             skuQueries,
		kwdQueries:             kwdQueries,
		lastKnownPid:           lastKnownPid,
		numAttemptsLoadProduct: 5,
		syncRequired:           false,
	}

	runCallback := func() {
		loadTask.loopMonitor()
	}
	stopCallback := func() {}

	baseTask, err := NewBaseTask(taskName, runCallback, stopCallback, proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task: %v", err)
	}

	loadTask.BaseTask = baseTask

	return loadTask, nil
}

func (t *LoadTask) addSkuQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	skuQuery := SkuQuery(strings.ToUpper(query))

	t.skuQueries = append(t.skuQueries, skuQuery)
}

func (t *LoadTask) addKeywordQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	query = strings.ToLower(query)
	q := createKeywordQuery(query)

	t.kwdQueries = append(t.kwdQueries, q)
}

func (t *LoadTask) removeSkuQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	skuQuery := SkuQuery(strings.ToUpper(query))

	for i, q := range t.skuQueries {
		if q == skuQuery {
			t.skuQueries = append(t.skuQueries[:i], t.skuQueries[i+1:]...)
			break
		}
	}
}

func (t *LoadTask) removeKeywordQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	query = strings.ToLower(query)
	q := createKeywordQuery(query)

	for i, kwdQuery := range t.kwdQueries {
		if kwdQuery.rawQueryStr == q.rawQueryStr {
			t.kwdQueries = append(t.kwdQueries[:i], t.kwdQueries[i+1:]...)
			break
		}
	}
}

func (t *LoadTask) loopMonitor() {
	configMu.RLock()
	defer time.Sleep(time.Millisecond * time.Duration(config.LoadTask.Timeout))
	configMu.RUnlock()

	t.rotateProxy()

	body, err := t.getNewArrivalsPage()
	if err != nil {
		t.logger.Red(err)
		return
	}

	// Initialize reference pid
	t.mu.Lock()
	if t.lastKnownPid == "" {
		t.syncRequired = true

		t.lastKnownPid, err = t.getFirstPid(body)
		if err != nil {
			t.logger.Red(fmt.Sprintf("Error getting initial pid: %v", err))

			t.mu.Unlock()
			return
		}
	}
	t.mu.Unlock()

	// Check for new products
	newProductUrls, err := t.findNewProductUrls(body)
	if err != nil {
		t.logger.Red(err)
		return
	}

	amountNewProducts := len(newProductUrls)
	if amountNewProducts > 0 {
		if amountNewProducts == 1 {
			t.logger.Gray("1 new product loaded. Checking for matches...")
		} else {
			t.logger.Gray(fmt.Sprintf("%d new products loaded. checking for matches...", amountNewProducts))
		}

		newestPid, err := t.getFirstPid(body)
		if err != nil {
			t.logger.Red(err)
			return
		}

		t.mu.Lock()
		if t.lastKnownPid != newestPid {
			t.syncRequired = true

			t.lastKnownPid = newestPid
		}
		t.mu.Unlock()

		// Check new products
		go func() {
			productsCh := make(chan *productWrapper)
			go t.wrapNewProducts(newProductUrls, productsCh)

			for newProduct := range productsCh {
				matchingSkuQueries := t.productMatchesSkuQueries(newProduct)
				matchingKwdQueries := t.productMatchesKeywordQueries(newProduct)

				matchingAny := len(matchingSkuQueries) > 0 || len(matchingKwdQueries) > 0

				if matchingAny {
					stateChanged := t.matchProductStates(newProduct, matchingSkuQueries, matchingKwdQueries)

					if stateChanged {
						t.mu.Lock()
						t.syncRequired = true
						t.mu.Unlock()
					} else {
						t.logger.Gray(fmt.Sprintf("No new match on product \"%s\"", newProduct.productData.ProductUrl))
					}
				} else {
					t.logger.Gray(fmt.Sprintf("No match on product \"%s\"", newProduct.productData.ProductUrl))
				}
			}
		}()
	} else {
		t.logger.Gray("No new products loaded")
	}

	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.syncRequired {
		t.syncRequired = false

		// Update last known pid
		productStates.Load.LastKnownPid = t.lastKnownPid

		go writeProductStates()
	}
}

func (t *LoadTask) getFirstPid(arrivalsPageBody ArrivalsPageBody) (string, error) {
	strArrivalsPageBody := string(arrivalsPageBody)

	if !strings.Contains(strArrivalsPageBody, "<div class=\"product-list\">") {
		return "", errors.New("get first pid: missing sequence start identifier")
	}
	strArrivalsPageBody = strings.Split(strArrivalsPageBody, "<div class=\"product-list\">")[1]

	if !strings.Contains(strArrivalsPageBody, "<article") {
		return "", errors.New("get first pid: missing products sequence start identifier")
	}
	firstProductSeq := strings.Split(strArrivalsPageBody, "<article")[1] // 1:  Get first product sequence

	if !strings.Contains(firstProductSeq, "data-item-id=\"") {
		return "", errors.New("get first pid: missing article pid sequence start identifier")
	}
	firstProductSeq = strings.Split(firstProductSeq, "data-item-id=\"")[1]

	if !strings.Contains(firstProductSeq, "\"") {
		return "", errors.New("get first pid: missing article pid sequence end identifier")
	}

	firstPid := strings.Split(firstProductSeq, "\"")[0]

	return firstPid, nil
}

func (t *LoadTask) findNewProductUrls(arrivalsPageBody ArrivalsPageBody) ([]string, error) {
	strArrivalsPageBody := string(arrivalsPageBody)

	if !strings.Contains(strArrivalsPageBody, "<div class=\"product-list\">") {
		return nil, errors.New("find new products: missing sequence start identifier")
	}
	strArrivalsPageBody = strings.Split(strArrivalsPageBody, "<div class=\"product-list\">")[1]

	if !strings.Contains(strArrivalsPageBody, "<article") {
		return nil, errors.New("find new products: missing products sequence start identifier")
	}
	productsRaw := strings.Split(strArrivalsPageBody, "<article")[1:]

	foundProducts := []string{}

	for _, product := range productsRaw {
		var pid string

		if !strings.Contains(product, "</article>") {
			return nil, errors.New("find new products: missing products sequence end identifier")
		}
		product = strings.Split(product, "</article>")[0]

		// find pid
		if !strings.Contains(product, "data-item-id=\"") {
			return nil, errors.New("find new products: missing article pid sequence start identifier")
		}
		pid = strings.Split(product, "data-item-id=\"")[1]

		if !strings.Contains(pid, "\"") {
			return nil, errors.New("find new products: missing article pid sequence end identifier")
		}
		pid = strings.Split(pid, "\"")[0]

		// Stop once we reached the last known pid
		if pid == t.lastKnownPid {
			break
		}

		// find product url
		if !strings.Contains(product, "<a href=\"") {
			return nil, errors.New("find new products: missing article product url sequence start identifier")
		}
		productUrl := strings.Split(product, "<a href=\"")[1]

		if !strings.Contains(productUrl, "\"") {
			return nil, errors.New("find new products: missing article product url sequence end identifier")
		}
		productUrl = strings.Split(productUrl, "\"")[0]

		foundProducts = append(foundProducts, productUrl)
	}

	return foundProducts, nil
}

func (t *LoadTask) wrapNewProducts(productUrls []string, ch chan *productWrapper) {
	wg := sync.WaitGroup{}

	for i, productUrl := range productUrls {
		wg.Add(1)
		go func() {
			for numAttempt := range t.numAttemptsLoadProduct {
				loadTask, err := NewLoadTask(fmt.Sprintf("%s.%d", t.taskName, i+1), t.proxyHandler, nil, []string{}, []string{}, "")
				if err != nil {
					t.logger.Red(fmt.Errorf("error creating load subtask %s: %v", fmt.Sprintf("%s.%d", t.taskName, i+1), err))
					return
				}

				loadTask.logger.Gray(fmt.Sprintf("Fetching new found product \"%s\"", productUrl))

				loadTask.rotateProxy()

				body, err := loadTask.getProductPage("https://www.sneakersnstuff.com" + strings.Split(productUrl, "\"")[0])
				if err != nil {
					loadTask.logger.Red(fmt.Sprintf("{%d/%d} find new products: getting product page: %v", numAttempt+1, t.numAttemptsLoadProduct, err))
					continue
				}

				// Create product data
				productData, err := t.createProductData(body, "https://www.sneakersnstuff.com"+productUrl)
				if err != nil {
					loadTask.logger.Red(fmt.Sprintf("{%d/%d} find new products: error creating product data: %v", numAttempt+1, t.numAttemptsLoadProduct, err))
					continue
				}

				loadTask.logger.Gray(fmt.Sprintf("New found product fetched \"%s\"", productUrl))

				// append to found product data
				product := &productWrapper{
					productData:      productData,
					productsPageBody: body,
				}

				ch <- product

				break
			}
			defer wg.Done()
		}()
	}

	wg.Wait()
	close(ch)
}

func (t *LoadTask) productMatchesSkuQueries(product *productWrapper) []SkuQuery {
	matchingQueries := []SkuQuery{}

	for _, skuQuery := range t.skuQueries {
		if skuQuery == SkuQuery(product.productData.Sku) {
			matchingQueries = append(matchingQueries, skuQuery)
		}
	}

	return matchingQueries
}

func (t *LoadTask) productMatchesKeywordQueries(product *productWrapper) []KwdQuery {
	productPageBodyStr := string(product.productsPageBody)

	color := ""
	supplierColor := ""

	if strings.Contains(productPageBodyStr, "\"color\":\"") {
		color = strings.Split(productPageBodyStr, "\"color\":\"")[1]
		if indexDelimiter := strings.Index(color, "\""); indexDelimiter > 0 {
			color = color[:indexDelimiter]
		} else {
			color = ""
		}
	}
	if strings.Contains(productPageBodyStr, "\"supplierColor\":\"") {
		supplierColor = strings.Split(productPageBodyStr, "\"supplierColor\":\"")[1]
		if indexDelimiter := strings.Index(supplierColor, "\""); indexDelimiter > 0 {
			supplierColor = supplierColor[:indexDelimiter]
		} else {
			supplierColor = ""
		}
	}

	productStr := strings.ToLower(fmt.Sprintf("%s %s %s", product.productData.Title, color, supplierColor))

	matchingQueries := []KwdQuery{}
	for _, kwdQuery := range t.kwdQueries {
		match := true
		// Inclusive keywords check
		for _, inclusiveKwd := range kwdQuery.inclusiveKeywords {
			if !strings.Contains(productStr, inclusiveKwd) {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		// Exclusive keywords check
		for _, exclusiveKwd := range kwdQuery.exclusiveKeywords {
			if strings.Contains(productStr, exclusiveKwd) {
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
				if strings.Contains(productStr, orKwd) {
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

		matchingQueries = append(matchingQueries, kwdQuery)
	}
	return matchingQueries
}

func (t *LoadTask) matchProductStates(product *productWrapper, matchingSkuQueries []SkuQuery, matchingKwdQueries []KwdQuery) bool {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	stateChange := false
	isNewProductUrl := true

	// Skus
	skuQueryStrings := []string{}
	for _, skuQuery := range matchingSkuQueries {
		skuQueryStrings = append(skuQueryStrings, string(skuQuery))
	}

	// Keywords
	keywordQueryStrings := []string{}
	for _, kwdQuery := range matchingKwdQueries {
		keywordQueryStrings = append(keywordQueryStrings, kwdQuery.rawQueryStr)
	}

	for _, productState := range productStates.Load.Products {
		if productState.ProductPageUrl == product.productData.ProductUrl {
			isNewProductUrl = false

			t.logger.Gray(fmt.Sprintf("Hit already notified product \"%s\"", product.productData.ProductUrl))

			productState.MatchingKeywordQueries = skuQueryStrings
			productState.MatchingKeywordQueries = keywordQueryStrings

			stateChange = true
			break
		}
	}

	if isNewProductUrl {
		t.logger.Green(fmt.Sprintf("Hit new product \"%s\"", product.productData.ProductUrl))

		productStateLoad := &ProductStateLoad{
			ProductPageUrl:         product.productData.ProductUrl,
			MatchingSkuQueries:     skuQueryStrings,
			MatchingKeywordQueries: keywordQueryStrings,
		}

		productStates.Load.Products = append(productStates.Load.Products, productStateLoad)

		stateChange = true

		t.NotifyLoad(product, skuQueryStrings, keywordQueryStrings)
	}

	return stateChange
}

func (t *LoadTask) NotifyLoad(product *productWrapper, matchingSkuQueries []string, matchingKwdQueries []string) {
	availableSizes, err := t.getAvailableSizes(product.productsPageBody)
	if err != nil {
		t.logger.Red(fmt.Sprintf("notify: product \"%s\": %v", product.productData.ProductUrl, err))
		return
	}
	price, err := t.getPrice(product.productsPageBody)
	if err != nil {
		t.logger.Red(fmt.Sprintf("notify: product \"%s\": %v", product.productData.ProductUrl, err))
		return
	}

	t.webhookHandler.NotifyLoad(product.productData, availableSizes, price, matchingSkuQueries, matchingKwdQueries)
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
