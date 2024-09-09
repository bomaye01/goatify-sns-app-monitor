package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	discordwebhook "github.com/bensch777/discord-webhook-golang"
)

type WebhookHandler struct {
	wg     sync.WaitGroup
	reqCh  chan *webhookRequest
	logger *Logger
}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		wg:     sync.WaitGroup{},
		reqCh:  make(chan *webhookRequest),
		logger: NewLogger("WEBHOOK"),
	}
}

type webhookRequest struct {
	productData *ProductData
	webhookUrl  string
	fields      []discordwebhook.Field
	time        time.Time
}

func (w *WebhookHandler) Start() {
	go func() {
		for req := range w.reqCh {
			embed, err := w.createEmbed(req)
			if err != nil {
				w.logger.Red(fmt.Sprintf("Create embed: %v", err))
				continue
			}

			w.sendEmbed(req.webhookUrl, embed)
		}
	}()
}

func (w *WebhookHandler) Stop() {
	w.wg.Wait()
	close(w.reqCh)
}

func (w *WebhookHandler) enqueueReq(req *webhookRequest) {
	w.wg.Add(1)
	go func() {
		w.reqCh <- req
		w.wg.Done()
	}()
}

func (w *WebhookHandler) NotifyRestock(productData *ProductData, availableSizes []string, price string) {
	configMu.RLock()
	defer configMu.RUnlock()

	for _, webhookUrl := range config.LoadTask.WebhookUrls {
		sizesValues := []string{}
		if len(availableSizes)-(25*len(sizesValues)) > 25 {
			sizesValues = append(sizesValues, strings.Join(availableSizes[:25], "\n"))
		} else {
			sizesValues = append(sizesValues, strings.Join(availableSizes, "\n"))
		}

		fields := []discordwebhook.Field{
			{
				Name:   "SKU/PID",
				Value:  productData.Sku,
				Inline: true,
			},
			{
				Name:   "PRICE",
				Value:  price,
				Inline: true,
			},
			{
				Name:   "TYPE",
				Value:  "RESTOCK",
				Inline: true,
			},
		}

		if len(sizesValues) == 0 {
			sizesField := discordwebhook.Field{
				Name:   "Sizes",
				Value:  "*none*",
				Inline: true,
			}
			fields = append(fields, sizesField)
		} else if len(sizesValues) == 1 {
			sizesField := discordwebhook.Field{
				Name:   "Sizes",
				Value:  sizesValues[0],
				Inline: true,
			}
			fields = append(fields, sizesField)
		} else {
			for i, sizesValue := range sizesValues {
				sizesField := discordwebhook.Field{
					Name:   fmt.Sprintf("Sizes %d", i+1),
					Value:  sizesValue,
					Inline: true,
				}
				fields = append(fields, sizesField)
			}
		}

		extraField := discordwebhook.Field{
			Name:   "Extra",
			Value:  fmt.Sprintf("[**StockX**](https://stockx.com/search?s=%s)", productData.Sku),
			Inline: false,
		}
		fields = append(fields, extraField)

		wreq := webhookRequest{
			productData: productData,
			time:        time.Now(),
			webhookUrl:  webhookUrl,
			fields:      fields,
		}

		w.enqueueReq(&wreq)
	}
}

func (w *WebhookHandler) NotifyPrice(productData *ProductData, availableSizes []string, oldPrice string, newPrice string) {
	configMu.RLock()
	defer configMu.RUnlock()

	for _, webhookUrl := range config.LoadTask.WebhookUrls {
		sizesValues := []string{}
		if len(availableSizes)-(25*len(sizesValues)) > 25 {
			sizesValues = append(sizesValues, strings.Join(availableSizes[:25], "\n"))
		} else {
			sizesValues = append(sizesValues, strings.Join(availableSizes, "\n"))
		}

		fields := []discordwebhook.Field{
			{
				Name:   "SKU/PID",
				Value:  productData.Sku,
				Inline: true,
			},
			{
				Name:   "PRICE",
				Value:  fmt.Sprintf("%s -> %s", oldPrice, newPrice),
				Inline: true,
			},
			{
				Name:   "TYPE",
				Value:  "PRICE CHANGE",
				Inline: true,
			},
		}

		if len(sizesValues) == 0 {
			sizesField := discordwebhook.Field{
				Name:   "Sizes",
				Value:  "*none*",
				Inline: true,
			}
			fields = append(fields, sizesField)
		} else if len(sizesValues) == 1 {
			sizesField := discordwebhook.Field{
				Name:   "Sizes",
				Value:  sizesValues[0],
				Inline: true,
			}
			fields = append(fields, sizesField)
		} else {
			for i, sizesValue := range sizesValues {
				sizesField := discordwebhook.Field{
					Name:   fmt.Sprintf("Sizes %d", i+1),
					Value:  sizesValue,
					Inline: true,
				}
				fields = append(fields, sizesField)
			}
		}

		extraField := discordwebhook.Field{
			Name:   "Extra",
			Value:  fmt.Sprintf("[**StockX**](https://stockx.com/search?s=%s)", productData.Sku),
			Inline: false,
		}
		fields = append(fields, extraField)

		wreq := webhookRequest{
			productData: productData,
			time:        time.Now(),
			webhookUrl:  webhookUrl,
			fields:      fields,
		}

		w.enqueueReq(&wreq)
	}
}

func (w *WebhookHandler) NotifyLoad(productData *ProductData, availableSizes []string, price string, matchingSkuQueries []string, matchingKwdQueries []string) {
	configMu.RLock()
	defer configMu.RUnlock()

	for _, webhookUrl := range config.LoadTask.WebhookUrls {
		sizesValues := []string{}
		if len(availableSizes)-(25*len(sizesValues)) > 25 {
			sizesValues = append(sizesValues, strings.Join(availableSizes[:25], "\n"))
		} else {
			sizesValues = append(sizesValues, strings.Join(availableSizes, "\n"))
		}

		fields := []discordwebhook.Field{
			{
				Name:   "SKU/PID",
				Value:  productData.Sku,
				Inline: true,
			},
			{
				Name:   "TYPE",
				Value:  "LOAD",
				Inline: true,
			},
			{
				Name:   "PRICE",
				Value:  price,
				Inline: true,
			},
		}

		if len(sizesValues) == 0 {
			sizesField := discordwebhook.Field{
				Name:   "Sizes",
				Value:  "*none*",
				Inline: true,
			}
			fields = append(fields, sizesField)
		} else if len(sizesValues) == 1 {
			sizesField := discordwebhook.Field{
				Name:   "Sizes",
				Value:  sizesValues[0],
				Inline: true,
			}
			fields = append(fields, sizesField)
		} else {
			for i, sizesValue := range sizesValues {
				sizesField := discordwebhook.Field{
					Name:   fmt.Sprintf("Sizes %d", i+1),
					Value:  sizesValue,
					Inline: true,
				}
				fields = append(fields, sizesField)
			}
		}

		if len(matchingSkuQueries) > 0 {
			overshoot := 0
			if len(matchingSkuQueries) >= 25 { // 25 lines is max allowed lines per field
				overshoot = 25 - len(matchingSkuQueries)
				matchingSkuQueries = matchingSkuQueries[:24]
			}
			strVal := strings.Join(matchingSkuQueries, "\n")
			if overshoot > 0 {
				strVal += fmt.Sprintf("*[+ %d more]*", overshoot)
			}

			skuQueryField := discordwebhook.Field{
				Name:   "SKU Query Hits",
				Value:  strVal,
				Inline: false,
			}
			fields = append(fields, skuQueryField)
		}
		if len(matchingKwdQueries) > 0 {
			overshoot := 0
			if len(matchingKwdQueries) >= 25 { // 25 lines is max allowed lines per field
				overshoot = 25 - len(matchingKwdQueries)
				matchingKwdQueries = matchingKwdQueries[:24]
			}
			strVal := strings.Join(matchingKwdQueries, "\n")
			if overshoot > 0 {
				strVal += fmt.Sprintf("*[+ %d more]*", overshoot)
			}

			kwdQueryField := discordwebhook.Field{
				Name:   "Keyword Query Hits",
				Value:  strVal,
				Inline: false,
			}
			fields = append(fields, kwdQueryField)
		}

		extraField := discordwebhook.Field{
			Name:   "Extra",
			Value:  fmt.Sprintf("[**StockX**](https://stockx.com/search?s=%s)", productData.Sku),
			Inline: false,
		}
		fields = append(fields, extraField)

		wreq := webhookRequest{
			productData: productData,
			time:        time.Now(),
			webhookUrl:  webhookUrl,
			fields:      fields,
		}

		w.enqueueReq(&wreq)
	}
}

func (w *WebhookHandler) createEmbed(req *webhookRequest) (*discordwebhook.Embed, error) {
	embed := discordwebhook.Embed{
		Title:     req.productData.Title,
		Color:     10104109,
		Url:       req.productData.ProductUrl,
		Timestamp: req.time,
		Thumbnail: discordwebhook.Thumbnail{
			Url: req.productData.ImageUrl,
		},
		Fields: req.fields,
		Footer: discordwebhook.Footer{
			Text:     "Goatify Bricks • Sneakersnstuff",
			Icon_url: "https://cdn.discordapp.com/attachments/1008077694409379962/1169271584805097543/0a4de578debc18ad1448c3bb14197df1.png?ex=66db0805&is=66d9b685&hm=10bcd145748d5ead3fe4ff8876b5ec7578830915862176d50374c43ea65e695f&",
		},
	}

	return &embed, nil
}

func (w *WebhookHandler) sendEmbed(link string, embed *discordwebhook.Embed) {
	hook := discordwebhook.Hook{
		Username:   "Sneakersnstuff",
		Avatar_url: "https://cdn.discordapp.com/attachments/1008077694409379962/1169271584805097543/0a4de578debc18ad1448c3bb14197df1.png?ex=66db0805&is=66d9b685&hm=10bcd145748d5ead3fe4ff8876b5ec7578830915862176d50374c43ea65e695f&",
		Embeds:     []discordwebhook.Embed{*embed},
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		w.logger.Red(fmt.Sprintf("Send webhook: Error marshalling payload: %v", err))
		return
	}

	req, err := http.NewRequest("POST", link, bytes.NewBuffer(payload))
	if err != nil {
		w.logger.Red(fmt.Sprintf("Send webhook: Error creating request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		w.logger.Red(fmt.Sprintf("Send webhook: Error sending request: %v", err))
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Red(fmt.Sprintf("Send webhook: Error reading response body: %v", err))
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		w.logger.Red(fmt.Sprintf("Send webhook: Unexpected response (%d): %s", resp.StatusCode, bodyText))
	}
	if resp.StatusCode == 429 {
		configMu.RLock()

		w.logger.Red(fmt.Sprintf("Send webhook: Rate limit reached. Trying again in %d milliseconds", config.WebhookErrorTimeout))

		time.Sleep(time.Millisecond * time.Duration(config.WebhookErrorTimeout))

		configMu.RUnlock()

		w.sendEmbed(link, embed)
	}

}
