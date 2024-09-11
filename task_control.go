package main

import (
	"fmt"
	"strings"
)

func handleAdd(addMessage *AddMessage) error {
	if map[string]bool{"PRODUCT": true, "SKU": true, "KWD_QUERY": true}[addMessage.InputType] {
		if addMessage.InputType == "SKU" {
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
	if map[string]bool{"SKU": true, "KWD_QUERY": true}[removeMessage.InputType] {
		if removeMessage.InputType == "SKU" {
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
	if map[string]bool{"SKU": true, "KWD_QUERY": true}[listMessage.InputType] {
		if listMessage.InputType == "SKU" {
			statesNormalMu.Lock()
			defer statesNormalMu.Unlock()

			skus := []string{}
			for _, state := range productStates.Normal.ProductStates {
				skus = append(skus, state.Sku)
			}

			return skus, nil
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

func addSkuQuery(skuQuery string) {
	normalTaskGroup.AddSkuQuery(skuQuery, ProductData{})
}

func addKwdQuery(kwdQuery string) {
	loadTaskGroup.AddKwdQuery(kwdQuery)
}

func removeSkuQuery(skuQuery string) {
	normalTaskGroup.RemoveSkuQuery(skuQuery)
}

func removeKwdQuery(kwdQuery string) {
	loadTaskGroup.RemoveKwdQuery(kwdQuery)
}

func checkSkuQueryMonitored(skuQuery string) bool {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	skuQuery = strings.ToUpper(strings.TrimSpace(skuQuery))

	for _, state := range productStates.Normal.ProductStates {
		if state.Sku == skuQuery {
			return true
		}
	}

	return false
}

func checkKwdQueryMonitored(kwdQuery string) bool {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	kwdQuery = strings.ToLower(strings.TrimSpace(kwdQuery))

	for _, kwd := range productStates.Load.KeywordQueries {
		if strings.ToLower(kwd) == kwdQuery {
			return true
		}
	}
	return false
}
