package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	http "github.com/bogdanfinn/fhttp"
)

type ProductsPageBody string
type ArrivalsPageBody string

type SnsTask struct {
	*BaseTask
}

func (t *SnsTask) isProductPage(url string) (bool, error) {
	body, err := t.getProductPage(url)
	if serr, ok := err.(*StatusCodeError); ok {
		if serr.statusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("check product: %v", err)
	}

	_, err = t.createProductData(ProductsPageBody(body), url)
	if err != nil {
		return false, fmt.Errorf("check product: %v", err)
	}
	return true, nil
}

func (t *SnsTask) getProductPage(url string) (ProductsPageBody, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("product page: error creating request: %v", err)
	}

	configMu.RLock()
	req.Header = http.Header{
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Accept-Language":           {"de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7"},
		"Priority":                  {"u=0, i"},
		"Referer":                   {"https://www.sneakersnstuff.com/de"},
		"Sec-Ch-Ua":                 {config.Fingerprint.Sec_ch_ua},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {"\"Windows\""},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"same-origin"},
		"Sec-Fetch-User":            {"?1"},
		"Upgrade-Insecure-Requests": {"1"},
		"User-Agent":                {config.Fingerprint.UserAgent},
	}
	configMu.RUnlock()

	res, err := t.httpClient.Do(req)
	if err != nil {
		requestErr := &RequestError{
			location: "product page",
			err:      err,
		}

		if t.proxy != nil {
			requestErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return "", requestErr
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		statusCodeErr := &StatusCodeError{
			location:   "product page",
			statusCode: res.StatusCode,
			statusText: res.Status,
		}

		if t.proxy != nil {
			if res.StatusCode == http.StatusForbidden {
				t.proxyHandler.ReportBadProxy(t.proxy)
			}

			statusCodeErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return "", statusCodeErr
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("product page: error reading body: %v", err)
	}

	return ProductsPageBody(bytes), nil
}

func (t *SnsTask) getNewArrivalsPage() (ArrivalsPageBody, error) {
	req, err := http.NewRequest("GET", "https://www.sneakersnstuff.com/de/1/new-arrivals?p=406887&orderBy=Published", nil)
	if err != nil {
		return "", fmt.Errorf("new arrivals page: error creating request: %v", err)
	}

	configMu.RLock()
	req.Header = http.Header{
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Accept-Language":           {"de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7"},
		"Cache-Control":             {"max-age=0"},
		"Priority":                  {"u=0, i"},
		"Referer":                   {"https://www.sneakersnstuff.com/de"},
		"Sec-Ch-Ua":                 {config.Fingerprint.Sec_ch_ua},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {"\"Windows\""},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"same-origin"},
		"Sec-Fetch-User":            {"?1"},
		"Upgrade-Insecure-Requests": {"1"},
		"User-Agent":                {config.Fingerprint.UserAgent},
	}
	configMu.RUnlock()

	res, err := t.httpClient.Do(req)
	if err != nil {
		requestErr := &RequestError{
			location: "new arrivals page",
			err:      err,
		}

		if t.proxy != nil {
			requestErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return "", requestErr
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		statusCodeErr := &StatusCodeError{
			location:   "new arrivals page",
			statusCode: res.StatusCode,
			statusText: res.Status,
		}

		if t.proxy != nil {
			if res.StatusCode == http.StatusForbidden {
				t.proxyHandler.ReportBadProxy(t.proxy)
			}

			statusCodeErr.proxyAsString = ProxyAsString(*t.proxy)
		}
		return "", statusCodeErr
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("new arrivals page: error reading body: %v", err)
	}

	return ArrivalsPageBody(bytes), nil
}

func (t *SnsTask) getAvailableSizes(productsPageBody ProductsPageBody) ([]string, error) {
	strProductsPageBody := string(productsPageBody)

	if !strings.Contains(strProductsPageBody, "id=\"product-size\"") {
		if strings.Contains(strProductsPageBody, "<span class=\"product-form__size-text\">") {
			oneSizeStr := strings.Split(strProductsPageBody, "<span class=\"product-form__size-text\">")[1]

			endIndex := strings.Index(oneSizeStr, "<")
			if endIndex < 0 {
				return nil, errors.New("failed to identify available product one-size sequence end")
			}
			oneSizeStr = oneSizeStr[:endIndex]

			return []string{oneSizeStr}, nil
		}

		return nil, errors.New("failed to identify available product sizes sequence start")
	}
	strProductsPageBody = strings.Split(strProductsPageBody, "id=\"product-size\"")[1]

	if !strings.Contains(strProductsPageBody, "</select>") {
		return nil, errors.New("failed to identify available product sizes sequence end")
	}
	strProductsPageBody = strings.Split(strProductsPageBody, "</select>")[0]

	availableSizes := []string{}

	if !strings.Contains(strProductsPageBody, "<option") {
		return nil, errors.New("failed to identify initial product variant sequence start")
	}

	optionSplit := strings.Split(strProductsPageBody, "<option")
	if len(optionSplit) < 2 {
		return nil, errors.New("failed to identify initial product variant sequence start 2")
	}
	optionSplit = optionSplit[1:]

	for _, variantSeqAvailable := range optionSplit {
		if !strings.Contains(strProductsPageBody, "/option>") {
			return nil, errors.New("failed to identify product variant sequence end")
		}
		variantSeqAvailable = strings.Split(variantSeqAvailable, "/option>")[0]

		if strings.Contains(variantSeqAvailable, "converted-size-size-eu\":\"") {
			variantSeqAvailable = strings.Split(variantSeqAvailable, "converted-size-size-eu\":\"")[1]

			indexDelimiter := strings.Index(variantSeqAvailable, "\"")
			if indexDelimiter == -1 {
				return nil, errors.New("failed to identify EU size format sequence end 1")
			}
			variantSeqAvailable = variantSeqAvailable[:indexDelimiter]

			availableSizes = append(availableSizes, fmt.Sprintf("EU%s", variantSeqAvailable))
		} else if strings.Contains(variantSeqAvailable, ">") {
			variantSeqAvailable = strings.Split(variantSeqAvailable, ">")[1]

			indexDelimiter := strings.Index(variantSeqAvailable, "<")
			if indexDelimiter == -1 {
				return nil, errors.New("failed to identify EU size format sequence end 2")
			}
			variantSeqAvailable = variantSeqAvailable[:indexDelimiter]

			if variantSeqAvailable != "" {
				availableSizes = append(availableSizes, variantSeqAvailable)
			}
		} else {
			return nil, errors.New("failed to identify EU size format sequence")
		}
	}

	return availableSizes, nil
}

func (t *SnsTask) getPrice(productsPageBody ProductsPageBody) (string, error) {
	strProductsPageBody := string(productsPageBody)

	if !strings.Contains(strProductsPageBody, "class=\"price") {
		return "", errors.New("failed to identify available product price identifier")
	}
	productPriceSeq := strings.Split(strProductsPageBody, "class=\"price")[1]

	if !strings.Contains(productPriceSeq, "data-value=\"") {
		return "", errors.New("failed to identify available product price sequence start")
	}
	productPriceSeq = strings.Split(productPriceSeq, "data-value=\"")[1]

	indexDelimiter := strings.Index(productPriceSeq, "\"")
	if indexDelimiter == -1 {
		return "", errors.New("failed to identify available product price sequence end")
	}
	productPriceSeq = productPriceSeq[:indexDelimiter]

	return productPriceSeq, nil
}

func (t *SnsTask) createProductData(productsPageBody ProductsPageBody, productPageUrl string) (*ProductData, error) {
	productData := &ProductData{}

	strProductsPageBody := string(productsPageBody)

	// Title
	if !strings.Contains(strProductsPageBody, "<title>") { // Old <span class=\"product-view__title-name\">
		return nil, errors.New("get product data (title): missing sequence start identifier")
	}
	titleSequence := strings.Split(strProductsPageBody, "<title>")[1]

	if !strings.Contains(titleSequence, " -") {
		return nil, errors.New("get product data (title): missing sequence end identifier")
	}
	productData.Title = strings.Split(titleSequence, " -")[0]

	// sku
	if !strings.Contains(strProductsPageBody, "'ArtNo': '") {
		return nil, errors.New("get product data (sku): missing sequence start identifier")
	}
	skuSequence := strings.Split(strProductsPageBody, "'ArtNo': '")[1]

	if !strings.Contains(skuSequence, "'") {
		return nil, errors.New("get product data (sku): missing sequence end identifier")
	}
	productData.Sku = strings.ToUpper(strings.Split(skuSequence, "'")[0])

	// pid
	if !strings.Contains(strProductsPageBody, "data-item-id=\"") {
		return nil, errors.New("find new products: missing article pid sequence start identifier")
	}
	productData.Pid = strings.Split(strProductsPageBody, "data-item-id=\"")[1]

	if !strings.Contains(productData.Pid, "\"") {
		return nil, errors.New("find new products: missing article pid sequence end identifier")
	}
	productData.Pid = strings.Split(productData.Pid, "\"")[0]

	// image: <div class="image-gallery__track"> then href=" end "
	if !strings.Contains(strProductsPageBody, "<div class=\"image-gallery__track\">") {
		return nil, errors.New("get product data (image): missing sequence start identifier 1")
	}
	imageSequence := strings.Split(strProductsPageBody, "<div class=\"image-gallery__track\">")[1]

	if !strings.Contains(imageSequence, "data-placeholder=\"") {
		return nil, errors.New("get product data (image): missing sequence start identifier 2")
	}
	imageSequence = strings.Split(imageSequence, "data-placeholder=\"")[1]

	if !strings.Contains(imageSequence, "\"") {
		return nil, errors.New("get product data (image): missing sequence end identifier")
	}
	imageSequence = "https://www.sneakersnstuff.com" + strings.Split(imageSequence, "\"")[0]

	productData.ImageUrl = imageSequence

	productData.ProductUrl = productPageUrl

	return productData, nil
}
