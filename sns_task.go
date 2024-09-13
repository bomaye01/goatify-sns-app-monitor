package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	http "github.com/bogdanfinn/fhttp"
)

const (
	NEW_ARRIVALS_URL      string = "https://core.dxpapi.com/api/v1/core/?request_type=search&fl=pid%2Cprice%2Ctitle%2Cbrand%2Cthumb_image%2Cdescription%2Csku%2Cavailability%2Ccategories_path%2Csub_brand%2Cbrand_color%2Ccolor%2Cproduct_level%2Cstyle%2Cgender%2Cseason%2Coriginal_price_usd%2Coriginal_price_eur%2Coriginal_price_gbp%2Coriginal_price_dkk%2Coriginal_price_sek%2Csignup_end_date%2Craffle_delayed%2Cprice_usd%2Cprice_eur%2Cprice_gbp%2Cprice_dkk%2Cprice_sek%2Cproduct_type%2Cproduct_group%2Cis_raffle%2Cdiscount_usd%2Cdiscount_eur%2Cdiscount_gbp%2Cdiscount_dkk%2Cdiscount_sek%2Ccustom_tag%2Cmarket_reference_eu%2Cmarket_reference_uk%2Cmarket_reference_us%2Csize_clothing_US%2Csize_clothing_EU%2Csize_clothing_UK%2Csize_clothing_JP%2Csize_shoes_US%2Csize_shoes_EU%2Csize_shoes_UK%2Csize_shoes_JP%2Cpublishing_date%2Craffle_finalized%2Cis_in_stock%2Ceu_category_ids%2Cuk_category_ids%2Cus_category_ids%2Crelease_date_eu%2Crelease_date_uk%2Crelease_date_us&account_id=7488&view_id=app_emea&domain_key=sneakersnstuff_de&q=31d26a6487e08357bd771619e894b0c6&search_type=category&url=https%3A%2F%2Fsneakersnstuff.com&fq=Category%3A%22Skate-Sneakers%22%20OR%20%22Basketball-Schuhe%22%20OR%20%22Court-Sneakers%22%20OR%20%22Retro%20Basketball-Schuhe%22%20OR%20%22Laufschuhe%22%20OR%20%22Schuhe%22%20OR%20%22Slides%20%26%20Sandalen%22%20OR%20%22Retro%20Runners%22%20OR%20%22Trail-Sneakers%22&sort=publishing_date%20desc&start=0&rows=40"
	PRODUCTS_BY_SKU_QUERY string = "\n    query FindBySkus($skus: [String!]!, $currencyCode: currencyCode!, $includeTax: Boolean!) {\n      site {\n        search {\n          searchProducts(filters: { productAttributes: [{ attribute: \"sku\", values: $skus }] }) {\n            products(first: 50) {\n              ...ProductListFragment\n              __typename\n            }\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n    }\n    \n    fragment CustomFieldsFragment on CustomFieldConnection {\n      edges {\n        node {\n          entityId\n          name\n          value\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    \n    fragment MetafieldFragment on Metafields {\n      id\n      key\n      value\n      __typename\n    }\n    \n    fragment MetafieldsFragment on MetafieldConnection {\n      edges {\n        node {\n          ...MetafieldFragment\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    \n    fragment ListProductFragment on Product {\n      id\n      entityId\n      name\n      sku\n      path\n      defaultImage {\n        url(height: 250, width: 250)\n        __typename\n      }\n      brand {\n        name\n        __typename\n      }\n      images(first: 50) {\n        edges {\n          node {\n            url(height: 2000, width: 2000)\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      customFields(first: 50) {\n        ...CustomFieldsFragment\n        __typename\n      }\n      metafields(namespace: \"sns_metafields\", first: 50) {\n        ...MetafieldsFragment\n        __typename\n      }\n      availabilityV2 {\n        status\n        description\n        ... on ProductPreOrder {\n          willBeReleasedAt {\n            utc\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      categories(first: 50) {\n        edges {\n          node {\n            metafields(namespace: \"sns_metafields\") {\n              edges {\n                node {\n                  entityId\n                  key\n                  value\n                  __typename\n                }\n                __typename\n              }\n              __typename\n            }\n            id\n            entityId\n            name\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      prices(includeTax: $includeTax, currencyCode: $currencyCode) {\n        price {\n          currencyCode\n          value\n          __typename\n        }\n        basePrice {\n          currencyCode\n          value\n          __typename\n        }\n        salePrice {\n          currencyCode\n          value\n          __typename\n        }\n        priceRange {\n          min {\n            currencyCode\n            value\n            __typename\n          }\n          max {\n            currencyCode\n            value\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      inventory {\n        isInStock\n        __typename\n      }\n      description\n      variants(first: 50) {\n        edges {\n          node {\n            entityId\n            id\n            sku\n            prices(currencyCode: $currencyCode, includeTax: $includeTax) {\n              basePrice {\n                currencyCode\n                value\n                __typename\n              }\n              price {\n                currencyCode\n                value\n                __typename\n              }\n              salePrice {\n                currencyCode\n                value\n                __typename\n              }\n              __typename\n            }\n            inventory {\n              byLocation(first: 50) {\n                edges {\n                  node {\n                    locationEntityId\n                    availableToSell\n                    warningLevel\n                    isInStock\n                    locationEntityTypeId\n                    locationEntityCode\n                    __typename\n                  }\n                  __typename\n                }\n                __typename\n              }\n              aggregated {\n                availableToSell\n              }\n              isInStock\n              __typename\n            }\n            productOptions(first: 50) {\n              edges {\n                node {\n                  entityId\n                  __typename\n                  displayName\n                  ... on MultipleChoiceOption {\n                    values(first: 50) {\n                      edges {\n                        node {\n                          entityId\n                          label\n                          __typename\n                        }\n                        __typename\n                      }\n                      __typename\n                    }\n                    __typename\n                  }\n                }\n                __typename\n              }\n              __typename\n            }\n            metafields(namespace: \"sns_metafields\", first: 50) {\n              ...MetafieldsFragment\n              __typename\n            }\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    \n    fragment ProductNodeFragment on ProductEdge {\n      node {\n        ...ListProductFragment\n        __typename\n      }\n      __typename\n    }\n    \n    fragment ProductListFragment on ProductConnection {\n      edges {\n        ...ProductNodeFragment\n        __typename\n      }\n      __typename\n    }\n    "
	USER_AGENT            string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.6533.100 Safari/537.36"
)

type SnsTask struct {
	*BaseTask
}

func (t *SnsTask) getNewArrivals() (*NewArrivalsResponse, error) {
	req, err := http.NewRequest("GET", NEW_ARRIVALS_URL, nil)
	if err != nil {
		return nil, fmt.Errorf("new arrivals: error creating request: %v", err)
	}

	req.Header = http.Header{
		"Host":            {"core.dxpapi.com"},
		"Accept":          {"application/json, text/plain, */*"},
		"Accept-Encoding": {"gzip, deflate, br"},
		"User-Agent":      {USER_AGENT},
		"Connection":      {"keep-alive"},
		"Header-Order:": {
			"Host",
			"BC-Instance",
			"Accept",
			"Accept-Encoding",
			"User-Agent",
			"Connection",
		},
	}

	res, err := t.httpClient.Do(req)
	if err != nil {
		requestErr := &RequestError{
			location: "new arrivals",
			err:      err,
		}

		if t.proxy != nil {
			requestErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return nil, requestErr
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		statusCodeErr := &StatusCodeError{
			location:   "new arrivals",
			statusCode: res.StatusCode,
			statusText: res.Status,
		}

		if t.proxy != nil {
			if res.StatusCode == http.StatusForbidden {
				t.proxyHandler.ReportBadProxy(t.proxy)
			}

			statusCodeErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return nil, statusCodeErr
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("new arrivals: error reading body: %v", err)
	}

	newArrivalsResponse := NewArrivalsResponse{}

	err = json.Unmarshal(bytes, &newArrivalsResponse)
	if err != nil {
		return nil, fmt.Errorf("new arrivals: error unmarshalling response bytes: %v", err)
	}

	return &newArrivalsResponse, nil
}

func (t *SnsTask) getProductsBySku(skus []string) (*ProductsBySkusResponse, error) {
	productsBySkuBody := productsBySkuBody{
		CurrencyCode: "EUR",
		Skus:         skus,
	}

	bodyBytes, err := json.Marshal(productsBySkuBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling products by sku body: %v", err)
	}

	req, err := http.NewRequest("POST", "https://www.sneakersnstuff.com/api/products-by-skus", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("products by sku: error creating request: %v", err)
	}

	req.Header = http.Header{
		"Host":               {"www.sneakersnstuff.com"},
		"Sec-Ch-Ua":          {"\"Chromium\";v=\"127\", \"Not)A;Brand\";v=\"99\""},
		"Sec-Ch-Ua-Platform": {"\"Windows\""},
		"Accept-Language":    {"de-DE"},
		"Sec-Ch-Ua-Mobile":   {"?0"},
		"User-Agent":         {USER_AGENT},
		"X-Recommendations":  {"true"},
		"Content-Type":       {"atext/plain;charset=UTF-8"},
		"Accept":             {"*/*"},
		"Origin":             {"https://www.sneakersnstuff.com"},
		"Sec-Fetch-Site":     {"same-origin"},
		"Sec-Fetch-Mode":     {"cors"},
		"Sec-Fetch-Dest":     {"empty"},
		"Referer":            {"https://www.sneakersnstuff.com/de/"},
		"Accept-Encoding":    {"gzip, deflate, br"},
		"Priority":           {"u=1, i"},
		"Header-Order:": {
			"Host",
			"Cookie",
			"Content-Length",
			"Sec-Ch-Ua",
			"Sec-Ch-Ua-Platform",
			"Accept-Language",
			"Sec-Ch-Ua-Mobile",
			"User-Agent",
			"X-Recommendations",
			"Content-Type",
			"Accept",
			"Origin",
			"Sec-Fetch-Site",
			"Sec-Fetch-Mode",
			"Sec-Fetch-Dest",
			"Referer",
			"Accept-Encoding",
			"Priority",
		},
	}

	res, err := t.httpClient.Do(req)
	if err != nil {
		requestErr := &RequestError{
			location: "products by sku",
			err:      err,
		}

		if t.proxy != nil {
			requestErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return nil, requestErr
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		statusCodeErr := &StatusCodeError{
			location:   "products by sku",
			statusCode: res.StatusCode,
			statusText: res.Status,
		}

		if t.proxy != nil {
			if res.StatusCode == http.StatusForbidden {
				t.proxyHandler.ReportBadProxy(t.proxy)
			}

			statusCodeErr.proxyAsString = ProxyAsString(*t.proxy)
		}

		return nil, statusCodeErr
	}

	resbytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("products by sku: error reading body: %v", err)
	}

	productsBySkusResponse := ProductsBySkusResponse{}

	err = json.Unmarshal(resbytes, &productsBySkusResponse)
	if err != nil {
		return nil, fmt.Errorf("products by sku: error unmarshalling response bytes: %v", err)
	}

	return &productsBySkusResponse, nil
}

func getChangesToAvailable(knownSizes []AvailableSize, newSizes []AvailableSize) []AvailableSize {
	sizesChangedToAvailable := []AvailableSize{}

	for _, newSize := range newSizes {
		included := false

		for _, knownSize := range knownSizes {
			if knownSize.Name == newSize.Name {
				included = true
				break
			}
		}

		if !included {
			sizesChangedToAvailable = append(sizesChangedToAvailable, newSize)
		}
	}

	return sizesChangedToAvailable
}

func GetProductData(productNode ProductNode) ProductData {
	sku := productNode.Sku
	productUrl := fmt.Sprintf("https://www.sneakersnstuff.com/de%s", productNode.Path)

	title := productNode.Name

	availableSizes := []AvailableSize{}
	sizesMetafieldExisting := false

	for _, variantEdge := range productNode.Variants.Edges {
		if variantEdge.Node.Inventory.Aggregated.AvailableToSell == 0 {
			continue
		}

		for _, metafieldEdge := range variantEdge.Node.Metafields.Edges {
			if metafieldEdge.Node.Key == "sizes" {
				sizesMetafieldExisting = true

				if strings.Contains(metafieldEdge.Node.Value, "\"EU\":\"") {
					euSizeValue := strings.Split(metafieldEdge.Node.Value, "\"EU\":\"")[1]

					if strings.Contains(euSizeValue, "\"") {
						euSizeValue = strings.Split(euSizeValue, "\"")[0]

						euSizeValue = fmt.Sprintf("EU %s", euSizeValue)

						availableSize := AvailableSize{
							Name:          euSizeValue,
							AmountInStock: variantEdge.Node.Inventory.Aggregated.AvailableToSell,
						}

						availableSizes = append(availableSizes, availableSize)
					}
				} else if strings.Contains(metafieldEdge.Node.Value, "\"US\":\"") {
					euSizeValue := strings.Split(metafieldEdge.Node.Value, "\"US\":\"")[1]

					if strings.Contains(euSizeValue, "\"") {
						euSizeValue = strings.Split(euSizeValue, "\"")[0]

						euSizeValue = fmt.Sprintf("US %s", euSizeValue)

						availableSize := AvailableSize{
							Name:          euSizeValue,
							AmountInStock: variantEdge.Node.Inventory.Aggregated.AvailableToSell,
						}

						availableSizes = append(availableSizes, availableSize)
					}
				} else if strings.Contains(metafieldEdge.Node.Value, "\"UK\":\"") {
					euSizeValue := strings.Split(metafieldEdge.Node.Value, "\"UK\":\"")[1]

					if strings.Contains(euSizeValue, "\"") {
						euSizeValue = strings.Split(euSizeValue, "\"")[0]

						euSizeValue = fmt.Sprintf("UK %s", euSizeValue)

						availableSize := AvailableSize{
							Name:          euSizeValue,
							AmountInStock: variantEdge.Node.Inventory.Aggregated.AvailableToSell,
						}

						availableSizes = append(availableSizes, availableSize)
					}
				}

				break
			}
		}
	}

	if !sizesMetafieldExisting && len(productNode.Variants.Edges) == 1 {
		availableSize := AvailableSize{
			Name:          "One-Size",
			AmountInStock: productNode.Variants.Edges[0].Node.Inventory.Aggregated.AvailableToSell,
		}

		availableSizes = append(availableSizes, availableSize)
	}

	sortAvailableSizes(availableSizes)

	price := strconv.Itoa(productNode.Prices.Price.Value)

	imageUrl := productNode.DefaultImage.URL

	identifyerStr := ""
	// for _, metafieldEdge := range productNode.Metafields.Edges {
	// 	if metafieldEdge.Node.Key == "product_copy" {
	// 		if strings.Contains(metafieldEdge.Node.Value, "\\r\\n \\r\\n- ") {
	// 			identifyerStr = strings.Split(metafieldEdge.Node.Value, "\\r\\n \\r\\n- ")[1]

	// 			if strings.Contains(identifyerStr, "\\r") {
	// 				identifyerStr = strings.Split(identifyerStr, "\\r")[0]
	// 			} else {
	// 				identifyerStr = ""
	// 			}
	// 		} else if strings.Contains(metafieldEdge.Node.Value, "\\r\\n\\r\\n-\u00a0") {
	// 			identifyerStr = strings.Split(metafieldEdge.Node.Value, "\\r\\n\\r\\n-\u00a0")[1]

	// 			if strings.Contains(identifyerStr, "\\r") {
	// 				identifyerStr = strings.Split(identifyerStr, "\\r")[0]
	// 			} else if strings.Contains(identifyerStr, "\u00a0") {
	// 				identifyerStr = strings.Split(identifyerStr, "\u00a0")[0]
	// 			} else {
	// 				identifyerStr = ""
	// 			}
	// 		}

	// 		break
	// 	}
	// }
	for _, customfieldEdge := range productNode.CustomFields.Edges {
		if customfieldEdge.Node.Name == "brand_color" {
			identifyerStr = fmt.Sprintf("%s %s %s %s", productNode.Brand.Name, strings.ReplaceAll(title, sku, ""), identifyerStr, customfieldEdge.Node.Value)

			break
		}
	}

	identifyerStr = strings.ReplaceAll(identifyerStr, "+", " ")
	identifyerStr = strings.ReplaceAll(identifyerStr, "-", " ")
	identifyerStr = strings.ReplaceAll(identifyerStr, "/", " ")
	identifyerStr = strings.ReplaceAll(identifyerStr, "|", " ")
	identifyerStr = strings.ToLower(identifyerStr)
	for strings.Contains(identifyerStr, "  ") {
		identifyerStr = strings.ReplaceAll(identifyerStr, "  ", " ")
	}

	return ProductData{
		ProductUrl:     productUrl,
		Title:          title,
		Sku:            sku,
		AvailableSizes: availableSizes,
		Price:          price,
		ImageUrl:       imageUrl,
		IdentifyerStr:  identifyerStr,
	}
}

func sortAvailableSizes(sizes []AvailableSize) {
	sort.Slice(sizes, func(i, j int) bool {
		return sizes[i].Name < sizes[j].Name
	})
}
