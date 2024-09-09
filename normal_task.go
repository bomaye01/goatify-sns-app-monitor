package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type NormalTask struct {
	*SnsTask
	productPageUrl string
	availableSizes []string
	price          string
}

func NewNormalTask(taskName string, productPageUrl string, availableSizes []string, price string, proxyHandler *ProxyHandler, webhookHandler *WebhookHandler) (*NormalTask, error) {
	normalTask := &NormalTask{
		SnsTask:        &SnsTask{},
		productPageUrl: productPageUrl,
		availableSizes: availableSizes,
		price:          price,
	}

	runCallback := func() {
		normalTask.loopMonitor()
	}
	stopCallback := func() {}

	baseTask, err := NewBaseTask(taskName, runCallback, stopCallback, proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task: %v", err)
	}

	normalTask.SnsTask.BaseTask = baseTask

	return normalTask, nil
}

func (t *NormalTask) loopMonitor() {
	configMu.RLock()
	defer time.Sleep(time.Millisecond * time.Duration(config.NormalTask.Timeout))
	configMu.RUnlock()

	t.rotateProxy()

	body, err := t.getProductPage(t.productPageUrl)
	if err != nil {
		t.logger.Red(err)
		return
	}

	productData, err := t.createProductData(body, t.productPageUrl)
	if err != nil {
		t.logger.Red(err)
		return
	}

	availableSizes, err := t.getAvailableSizes(body)
	if err != nil {
		t.logger.Red(err)
		return
	}

	currentPrice, err := t.getPrice(body)
	if err != nil {
		t.logger.Red(err)
		return
	}

	// Update product properties
	t.mu.Lock()
	t.availableSizes = availableSizes
	t.price = currentPrice
	t.mu.Unlock()

	// Compare new data with known product states
	stateChanged := t.matchProductStates(productData)
	if stateChanged {
		go writeProductStates()
	}
}

func (t *NormalTask) getChangesToAvailable(knownAvailableSizes []string) []string {
	sizesChangedToAvailable := []string{}

	for _, currentAvailableSize := range t.availableSizes {
		included := false

		for _, knownAvailableSize := range knownAvailableSizes {
			if knownAvailableSize == currentAvailableSize {
				included = true
				break
			}
		}

		if !included {
			sizesChangedToAvailable = append(sizesChangedToAvailable, currentAvailableSize)
		}
	}

	return sizesChangedToAvailable
}

func (t *NormalTask) matchProductStates(productData *ProductData) bool {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()

	stateChange := false

	notifySize := false
	notifyPrice := false

	newAvailableSizes := []string{}
	oldPrice := ""

	for _, productState := range productStates.Normal.Products {
		if productState.ProductPageUrl == t.productPageUrl {
			if !reflect.DeepEqual(productState.AvailableSizes, t.availableSizes) {
				stateChange = true

				newAvailableSizes = t.getChangesToAvailable(productState.AvailableSizes)
				if len(newAvailableSizes) != 0 {
					notifySize = true
				}

				productState.AvailableSizes = t.availableSizes
			}

			if productState.Price != t.price {
				stateChange = true

				if t.price < productState.Price {
					notifyPrice = true
				}

				oldPrice = productState.Price

				productState.Price = t.price
			}
		}
	}

	// Console print
	if notifySize && notifyPrice {
		t.logger.Green(fmt.Sprintf("%s: New available sizes: %v | Price changed: %s -> %s", strings.Split(t.productPageUrl, "product/")[1], newAvailableSizes, oldPrice, t.price))
	} else if notifyPrice {
		if oldPrice > t.price {
			t.logger.Green(fmt.Sprintf("%s: Price changed: %s -> %s", strings.Split(t.productPageUrl, "product/")[1], oldPrice, t.price))
		} else {
			t.logger.Gray(fmt.Sprintf("%s: Price changed: %s -> %s", strings.Split(t.productPageUrl, "product/")[1], oldPrice, t.price))
		}
	} else if notifySize {
		t.logger.Green(fmt.Sprintf("%s: New available sizes: %v", strings.Split(t.productPageUrl, "product/")[1], newAvailableSizes))
	} else {
		t.logger.Gray(fmt.Sprintf("%s: No changes on product", strings.Split(t.productPageUrl, "product/")[1]))
	}

	// Webhook notify
	if notifySize {
		t.notifySize(productData)
	}
	if notifyPrice {
		if oldPrice > t.price {
			t.notifyPrice(productData, oldPrice, t.price)
		}
	}

	return stateChange
}

func (t *NormalTask) notifySize(productData *ProductData) {
	t.webhookHandler.NotifyRestock(productData, t.availableSizes, t.price)
}

func (t *NormalTask) notifyPrice(productData *ProductData, oldPrice string, newPrice string) {
	t.webhookHandler.NotifyPrice(productData, t.availableSizes, oldPrice, newPrice)
}
