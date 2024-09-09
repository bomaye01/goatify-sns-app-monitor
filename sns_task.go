package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	http "github.com/bogdanfinn/fhttp"
)

const (
	NEW_ARRIVALS_URL string = "https://core.dxpapi.com/api/v1/core/?request_type=search&fl=pid%2Cprice%2Ctitle%2Cbrand%2Cthumb_image%2Cdescription%2Csku%2Cavailability%2Ccategories_path%2Csub_brand%2Cbrand_color%2Ccolor%2Cproduct_level%2Cstyle%2Cgender%2Cseason%2Coriginal_price_usd%2Coriginal_price_eur%2Coriginal_price_gbp%2Coriginal_price_dkk%2Coriginal_price_sek%2Csignup_end_date%2Craffle_delayed%2Cprice_usd%2Cprice_eur%2Cprice_gbp%2Cprice_dkk%2Cprice_sek%2Cproduct_type%2Cproduct_group%2Cis_raffle%2Cdiscount_usd%2Cdiscount_eur%2Cdiscount_gbp%2Cdiscount_dkk%2Cdiscount_sek%2Ccustom_tag%2Cmarket_reference_eu%2Cmarket_reference_uk%2Cmarket_reference_us%2Csize_clothing_US%2Csize_clothing_EU%2Csize_clothing_UK%2Csize_clothing_JP%2Csize_shoes_US%2Csize_shoes_EU%2Csize_shoes_UK%2Csize_shoes_JP%2Cpublishing_date%2Craffle_finalized%2Cis_in_stock%2Ceu_category_ids%2Cuk_category_ids%2Cus_category_ids%2Crelease_date_eu%2Crelease_date_uk31d26a6487e08357bd771619e894b0c6%2Crelease_date_us&account_id=7488&view_id=app_emea&domain_key=sneakersnstuff_de&q=31d26a6487e08357bd771619e894b0c6&search_type=category&url=https%3A%2F%2Fsneakersnstuff.com&sort=publishing_date%20desc&start=0&rows=40"
	USER_AGENT       string = "okhttp/4.12.0"
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

func (t *SnsTask) getProductsBySku() (*ProductsBySkusResponse, error) {
	req, err := http.NewRequest("GET", NEW_ARRIVALS_URL, nil)
	if err != nil {
		return nil, fmt.Errorf("products by sku: error creating request: %v", err)
	}

	body := ""

	req.Header = http.Header{
		"Host":            {"app-api.sneakersnstuffapp.com"},
		"BC-Instance":     {"EU"},
		"Accept":          {"application/json"},
		"Content-Type":    {"application/json"},
		"Content-Length":  {strconv.Itoa(len(body))},
		"Accept-Encoding": {"gzip, deflate, br"},
		"User-Agent":      {USER_AGENT},
		"Header-Order:": {
			"Host",
			"BC-Instance",
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"User-Agent",
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

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("products by sku: error reading body: %v", err)
	}

	productsBySkusResponse := ProductsBySkusResponse{}

	err = json.Unmarshal(bytes, &productsBySkusResponse)
	if err != nil {
		return nil, fmt.Errorf("products by sku: error unmarshalling response bytes: %v", err)
	}

	return &productsBySkusResponse, nil
}
