package main

// req: new arrivals
type NewArrivalsResponse struct {
	Response Response `json:"response"`
}

type Response struct {
	ProductNodes []ProductNodeReference `json:"docs"`
}

type ProductNodeReference struct {
	ProductType string `json:"product_type"`
	Sku         string `json:"sku"`
	Pid         string `json:"pid"`
}

// req: products by skus

type productsBySkuBody struct {
	CurrencyCode string   `json:"currency"`
	Skus         []string `json:"skus"`
}

type ProductsBySkusResponse []ProductNode

type ProductNode struct {
	Sku      string `json:"sku"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	EntityID int    `json:"entityId"`
	Brand    struct {
		Name string `json:"name"`
	} `json:"brand"`
	AvailabilityV2 struct {
		Status string `json:"status"`
	} `json:"availabilityV2"`
	Prices struct {
		BasePrice struct {
			CurrencyCode string `json:"currencyCode"`
			Value        int    `json:"value"`
		} `json:"basePrice"`
		Price struct {
			CurrencyCode string `json:"currencyCode"`
			Value        int    `json:"value"`
		} `json:"price"`
		SalePrice interface{} `json:"salePrice"`
	} `json:"prices"`
	Variants struct {
		Edges []struct {
			Node struct {
				EntityID  int `json:"entityId"`
				Inventory struct {
					IsInStock  bool `json:"isInStock"`
					Aggregated struct {
						AvailableToSell int `json:"availableToSell"`
					} `json:"aggregated"`
				} `json:"inventory"`
				Options struct {
					Edges []struct {
						Node struct {
							Values struct {
								Edges []struct {
									Node struct {
										Label string `json:"label"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"values"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"options"`
				Metafields struct {
					Edges []struct {
						Node struct {
							Key   string `json:"key"`
							Value string `json:"value"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"metafields"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"variants"`
	DefaultImage struct {
		URLOriginal string `json:"urlOriginal"`
		URL460W     string `json:"url_460w"`
		URL         string `json:"url"`
	} `json:"defaultImage"`
	CustomFields struct {
		Edges []struct {
			Node struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"customFields"`
}
