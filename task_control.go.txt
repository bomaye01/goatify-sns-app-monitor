package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

func handleAdd(addMessage *AddMessage) error {
	if map[string]bool{"PRODUCT": true, "SKU_QUERY": true, "KWD_QUERY": true}[addMessage.InputType] {
		if addMessage.InputType == "PRODUCT" {
			qIndex := strings.Index(addMessage.AddQuery, "?")
			if qIndex >= 0 {
				addMessage.AddQuery = addMessage.AddQuery[:qIndex]
			}
			addMessage.AddQuery = strings.ToLower(addMessage.AddQuery)

			if checkProductPageAlreadyMonitored(addMessage.AddQuery) {
				return &AlreadyMonitoredError{
					queryType:  "PRODUCT",
					queryValue: addMessage.AddQuery,
				}
			}

			err := addProductTasks(addMessage.AddQuery)
			if err != nil {
				return err
			}
		} else if addMessage.InputType == "SKU_QUERY" {
			monitored := checkSkuQueryMonitored(addMessage.AddQuery)
			if monitored {
				return &AlreadyMonitoredError{
					queryType:  "SKU",
					queryValue: addMessage.AddQuery,
				}
			}

			addSkuQuery(addMessage.AddQuery)
		} else {
			if addMessage.AddQuery[0] != '+' && addMessage.AddQuery[0] != '-' {
				addMessage.AddQuery = fmt.Sprintf("+%s", addMessage.AddQuery)
			}

			monitored := checkKwdQueryMonitored(addMessage.AddQuery)
			if monitored {
				return &AlreadyMonitoredError{
					queryType:  "KEYWORD",
					queryValue: addMessage.AddQuery,
				}
			}

			addKwdQuery(addMessage.AddQuery)
		}
	} else {
		return fmt.Errorf("unexpected input type: %s", addMessage.InputType)
	}
	return nil
}

func handleRemove(removeMessage *RemoveMessage) error {
	if map[string]bool{"PRODUCT": true, "SKU_QUERY": true, "KWD_QUERY": true}[removeMessage.InputType] {
		if removeMessage.InputType == "PRODUCT" {
			qIndex := strings.Index(removeMessage.RemoveQuery, "?")
			if qIndex >= 0 {
				removeMessage.RemoveQuery = removeMessage.RemoveQuery[:qIndex]
			}
			removeMessage.RemoveQuery = strings.ToLower(removeMessage.RemoveQuery)

			err := removeProductTasks(removeMessage.RemoveQuery)
			if err != nil {
				return err
			}
		} else if removeMessage.InputType == "SKU_QUERY" {
			monitored := checkSkuQueryMonitored(removeMessage.RemoveQuery)
			if !monitored {
				return &QueryNotFoundError{
					queryType:  "SKU",
					queryValue: removeMessage.RemoveQuery,
				}
			}

			removeSkuQuery(removeMessage.RemoveQuery)
		} else {
			if removeMessage.RemoveQuery[0] != '+' && removeMessage.RemoveQuery[0] != '-' {
				removeMessage.RemoveQuery = fmt.Sprintf("+%s", removeMessage.RemoveQuery)
			}

			monitored := checkKwdQueryMonitored(removeMessage.RemoveQuery)
			if !monitored {
				return &QueryNotFoundError{
					queryType:  "KEYWORD",
					queryValue: removeMessage.RemoveQuery,
				}
			}

			removeKwdQuery(removeMessage.RemoveQuery)
		}
	} else {
		return fmt.Errorf("unexpected input type: %s", removeMessage.InputType)
	}
	return nil
}

func handleList(listMessage *ListMessage) ([]string, error) {
	if map[string]bool{"PRODUCTS": true, "SKU_QUERIES": true, "KWD_QUERIES": true}[listMessage.InputType] {
		if listMessage.InputType == "PRODUCTS" {
			productUrls := []string{}

			statesNormalMu.Lock()
			defer statesNormalMu.Unlock()

			for _, product := range productStates.Normal.Products {
				productUrls = append(productUrls, product.ProductPageUrl)
			}

			return productUrls, nil
		} else if listMessage.InputType == "SKU_QUERIES" {
			statesLoadMu.Lock()
			defer statesLoadMu.Unlock()

			queries := make([]string, len(productStates.Load.SkuQueries))
			copy(queries, productStates.Load.SkuQueries)

			return queries, nil
		} else {
			statesLoadMu.Lock()
			defer statesLoadMu.Unlock()

			queries := make([]string, len(productStates.Load.KeywordQueries))
			copy(queries, productStates.Load.KeywordQueries)

			return queries, nil
		}
	} else {
		return []string{}, fmt.Errorf("unexpected input type: %s", listMessage.InputType)
	}
}

func addProductTasks(productUrl string) error {
	tasksWg.Add(1)

	taskCountMu.Lock()
	normalTaskCount += 1

	taskName := fmt.Sprintf(" TEST : %02d", normalTaskCount)
	taskCountMu.Unlock()

	checkTask, err := NewNormalTask(taskName, productUrl, []string{}, "", proxyHandler, webhookHandler)
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error creating normal check task %s: %v", taskName, err))
	}

	errCh := make(chan error)

	go func() {
		defer close(errCh)

		testAttempts := 5
		isProductPage := false

		for i := range testAttempts {
			checkTask.rotateProxy()

			result, err := checkTask.isProductPage(productUrl)
			if err != nil {
				checkTask.logger.Red(fmt.Sprintf("{%d/%d} %v", i+1, testAttempts, err))
			} else if result {
				isProductPage = true
				break
			}
		}

		tasksWg.Done()

		taskCountMu.Lock()
		normalTaskCount -= 1
		taskCountMu.Unlock()

		if isProductPage {
			mainLogger.Cyan(fmt.Sprintf("Product url \"%s\" valid. Starting tasks...", productUrl))

			configMu.RLock()
			statesNormalMu.Lock()

			newProductState := &ProductStateNormal{
				ProductPageUrl: productUrl,
			}
			productStates.Normal.Products = append(productStates.Normal.Products, newProductState)

			statesNormalMu.Unlock()

			for range config.NormalTask.NumTasksPerProduct {
				tasksWg.Add(1)

				taskCountMu.Lock()
				normalTaskCount += 1

				taskName := fmt.Sprintf("NORMAL: %02d", normalTaskCount)
				taskCountMu.Unlock()

				normalTask, err := NewNormalTask(taskName, productUrl, []string{}, "", proxyHandler, webhookHandler)
				if err != nil {
					mainLogger.Red(fmt.Sprintf("Error creating normal added task %s: %v", taskName, err))
				}

				taskReferenceMu.Lock()
				normalTasksByProductUrl[normalTask.productPageUrl] = append(normalTasksByProductUrl[normalTask.productPageUrl], normalTask)
				taskReferenceMu.Unlock()

				go func() {
					if config.NormalTask.BurstStart {
						offsetMilliseconds := rand.Intn(config.NormalTask.Timeout)
						time.Sleep(time.Millisecond * time.Duration(offsetMilliseconds))
					}

					normalTask.Start()

					normalTask.WaitForTermination()

					tasksWg.Done()

					taskCountMu.Lock()
					normalTaskCount -= 1
					taskCountMu.Unlock()
				}()
			}

			configMu.RUnlock()
		} else {
			errCh <- &NotAProductPageError{
				url: productUrl,
			}
		}
	}()

	for err := range errCh {
		return err
	}

	return nil
}

func addSkuQuery(skuQuery string) {
	taskReferenceMu.Lock()

	for _, loadTask := range loadTasks {
		loadTask.addSkuQuery(skuQuery)
	}

	taskReferenceMu.Unlock()

	statesLoadMu.Lock()

	productStates.Load.SkuQueries = append(productStates.Load.SkuQueries, skuQuery)

	statesLoadMu.Unlock()

	go writeProductStates()
}

func addKwdQuery(kwdQuery string) {
	kwdQuery = strings.ToLower(kwdQuery)

	taskReferenceMu.Lock()

	for _, loadTask := range loadTasks {
		loadTask.addKeywordQuery(kwdQuery)
	}

	taskReferenceMu.Unlock()

	statesLoadMu.Lock()

	productStates.Load.KeywordQueries = append(productStates.Load.KeywordQueries, kwdQuery)

	statesLoadMu.Unlock()

	go writeProductStates()
}

func removeSkuQuery(skuQuery string) {
	skuQuery = strings.ToUpper(skuQuery)

	taskReferenceMu.Lock()

	for _, loadTask := range loadTasks {
		loadTask.removeSkuQuery(skuQuery)
	}

	taskReferenceMu.Unlock()

	statesLoadMu.Lock()

	qIndex := -1
	for i, query := range productStates.Load.SkuQueries {
		if strings.ToUpper(query) == skuQuery {
			qIndex = i
			break
		}
	}

	if qIndex >= 0 {
		productStates.Load.SkuQueries = append(productStates.Load.SkuQueries[:qIndex], productStates.Load.SkuQueries[qIndex+1:]...)
	}

	statesLoadMu.Unlock()

	go writeProductStates()
}

func removeKwdQuery(kwdQuery string) {
	kwdQuery = strings.ToLower(kwdQuery)

	taskReferenceMu.Lock()

	for _, loadTask := range loadTasks {
		loadTask.removeKeywordQuery(kwdQuery)
	}

	taskReferenceMu.Unlock()

	statesLoadMu.Lock()

	qIndex := -1
	for i, query := range productStates.Load.KeywordQueries {
		if strings.ToLower(query) == kwdQuery {
			qIndex = i
			break
		}
	}

	if qIndex >= 0 {
		productStates.Load.KeywordQueries = append(productStates.Load.KeywordQueries[:qIndex], productStates.Load.KeywordQueries[qIndex+1:]...)
	}

	statesLoadMu.Unlock()

	go writeProductStates()
}

func removeProductTasks(productUrl string) error {
	taskReferenceMu.Lock()
	defer taskReferenceMu.Unlock()

	tasks := normalTasksByProductUrl[productUrl]

	if len(tasks) == 0 {
		return &QueryNotFoundError{
			queryType:  "PRODUCT",
			queryValue: productUrl,
		}
	}

	wg := sync.WaitGroup{}

	for _, task := range tasks {
		wg.Add(1)
		go func() {
			task.Stop()
			task.WaitForTermination()

			wg.Done()
		}()
	}

	statesNormalMu.Lock()

	productUrl = strings.TrimPrefix(productUrl, "https://")
	productUrl = strings.TrimPrefix(productUrl, "www.")

	removeIndex := -1
	for i, product := range productStates.Normal.Products {
		sProductUrl := product.ProductPageUrl
		sProductUrl = strings.TrimPrefix(sProductUrl, "https://")
		sProductUrl = strings.TrimPrefix(sProductUrl, "www.")

		if sProductUrl == productUrl {
			removeIndex = i
			break
		}
	}

	if removeIndex >= 0 {
		productStates.Normal.Products = append(productStates.Normal.Products[:removeIndex], productStates.Normal.Products[removeIndex+1:]...)
	}

	statesNormalMu.Unlock()

	go writeProductStates()

	wg.Wait()
	return nil
}

func checkProductPageAlreadyMonitored(productUrl string) bool {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	productUrl = strings.TrimPrefix(productUrl, "https://")
	productUrl = strings.TrimPrefix(productUrl, "www.")

	for _, product := range productStates.Normal.Products {
		sProductUrl := product.ProductPageUrl
		sProductUrl = strings.TrimPrefix(sProductUrl, "https://")
		sProductUrl = strings.TrimPrefix(sProductUrl, "www.")

		if sProductUrl == productUrl {
			return true
		}
	}
	return false
}

func checkSkuQueryMonitored(skuQuery string) bool {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	skuQuery = strings.ToUpper(skuQuery)

	for _, sku := range productStates.Load.SkuQueries {
		if strings.ToUpper(sku) == skuQuery {
			return true
		}
	}

	return false
}

func checkKwdQueryMonitored(kwdQuery string) bool {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	kwdQuery = strings.ToLower(kwdQuery)

	for _, kwd := range productStates.Load.KeywordQueries {
		if strings.ToLower(kwd) == kwdQuery {
			return true
		}
	}
	return false
}
