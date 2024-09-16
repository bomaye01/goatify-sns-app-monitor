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
	Query     string        `json:"query"`
	Variables bodyVariables `json:"variables"`
}

type bodyVariables struct {
	Skus         []string `json:"skus"`
	CurrencyCode string   `json:"currencyCode"`
	IncludeTax   bool     `json:"includeTax"`
}

type ProductsBySkusResponse struct {
	Data struct {
		Site struct {
			Search struct {
				SearchProducts struct {
					Products struct {
						Edges []struct {
							Node     ProductNode `json:"node"`
							Typename string      `json:"__typename"`
						} `json:"edges"`
						Typename string `json:"__typename"`
					} `json:"products"`
					Typename string `json:"__typename"`
				} `json:"searchProducts"`
				Typename string `json:"__typename"`
			} `json:"search"`
			Typename string `json:"__typename"`
		} `json:"site"`
	} `json:"data"`
}

type ProductNode struct {
	ID           string `json:"id"`
	EntityID     int    `json:"entityId"`
	Name         string `json:"name"`
	Sku          string `json:"sku"`
	Path         string `json:"path"`
	DefaultImage struct {
		URL      string `json:"url"`
		Typename string `json:"__typename"`
	} `json:"defaultImage"`
	Brand struct {
		Name     string `json:"name"`
		Typename string `json:"__typename"`
	} `json:"brand"`
	Images struct {
		Edges []struct {
			Node struct {
				URL      string `json:"url"`
				Typename string `json:"__typename"`
			} `json:"node"`
			Typename string `json:"__typename"`
		} `json:"edges"`
		Typename string `json:"__typename"`
	} `json:"images"`
	CustomFields struct {
		Edges []struct {
			Node struct {
				EntityID int    `json:"entityId"`
				Name     string `json:"name"`
				Value    string `json:"value"`
				Typename string `json:"__typename"`
			} `json:"node"`
			Typename string `json:"__typename"`
		} `json:"edges"`
		Typename string `json:"__typename"`
	} `json:"customFields"`
	Metafields struct {
		Edges []struct {
			Node struct {
				ID       string `json:"id"`
				Key      string `json:"key"`
				Value    string `json:"value"`
				Typename string `json:"__typename"`
			} `json:"node"`
			Typename string `json:"__typename"`
		} `json:"edges"`
		Typename string `json:"__typename"`
	} `json:"metafields"`
	AvailabilityV2 struct {
		Status      string `json:"status"`
		Description string `json:"description"`
		Typename    string `json:"__typename"`
	} `json:"availabilityV2"`
	Categories struct {
		Edges []struct {
			Node struct {
				Metafields struct {
					Edges []struct {
						Node struct {
							EntityID int    `json:"entityId"`
							Key      string `json:"key"`
							Value    string `json:"value"`
							Typename string `json:"__typename"`
						} `json:"node"`
						Typename string `json:"__typename"`
					} `json:"edges"`
					Typename string `json:"__typename"`
				} `json:"metafields"`
				ID       string `json:"id"`
				EntityID int    `json:"entityId"`
				Name     string `json:"name"`
				Typename string `json:"__typename"`
			} `json:"node"`
			Typename string `json:"__typename"`
		} `json:"edges"`
		Typename string `json:"__typename"`
	} `json:"categories"`
	Prices struct {
		Price struct {
			CurrencyCode string `json:"currencyCode"`
			Value        int    `json:"value"`
			Typename     string `json:"__typename"`
		} `json:"price"`
		BasePrice struct {
			CurrencyCode string `json:"currencyCode"`
			Value        int    `json:"value"`
			Typename     string `json:"__typename"`
		} `json:"basePrice"`
		SalePrice  interface{} `json:"salePrice"`
		PriceRange struct {
			Min struct {
				CurrencyCode string `json:"currencyCode"`
				Value        int    `json:"value"`
				Typename     string `json:"__typename"`
			} `json:"min"`
			Max struct {
				CurrencyCode string `json:"currencyCode"`
				Value        int    `json:"value"`
				Typename     string `json:"__typename"`
			} `json:"max"`
			Typename string `json:"__typename"`
		} `json:"priceRange"`
		Typename string `json:"__typename"`
	} `json:"prices"`
	Inventory struct {
		IsInStock bool   `json:"isInStock"`
		Typename  string `json:"__typename"`
	} `json:"inventory"`
	Description string `json:"description"`
	Variants    struct {
		Edges []struct {
			Node struct {
				EntityID int    `json:"entityId"`
				ID       string `json:"id"`
				Sku      string `json:"sku"`
				Prices   struct {
					BasePrice struct {
						CurrencyCode string `json:"currencyCode"`
						Value        int    `json:"value"`
						Typename     string `json:"__typename"`
					} `json:"basePrice"`
					Price struct {
						CurrencyCode string `json:"currencyCode"`
						Value        int    `json:"value"`
						Typename     string `json:"__typename"`
					} `json:"price"`
					SalePrice interface{} `json:"salePrice"`
					Typename  string      `json:"__typename"`
				} `json:"prices"`
				Inventory struct {
					ByLocation struct {
						Edges    []interface{} `json:"edges"`
						Typename string        `json:"__typename"`
					} `json:"byLocation"`
					Aggregated struct {
						AvailableToSell int `json:"availableToSell"`
					} `json:"aggregated"`
					IsInStock bool   `json:"isInStock"`
					Typename  string `json:"__typename"`
				} `json:"inventory"`
				ProductOptions struct {
					Edges []struct {
						Node struct {
							EntityID    int    `json:"entityId"`
							Typename    string `json:"__typename"`
							DisplayName string `json:"displayName"`
							Values      struct {
								Edges []struct {
									Node struct {
										EntityID int    `json:"entityId"`
										Label    string `json:"label"`
										Typename string `json:"__typename"`
									} `json:"node"`
									Typename string `json:"__typename"`
								} `json:"edges"`
								Typename string `json:"__typename"`
							} `json:"values"`
						} `json:"node"`
						Typename string `json:"__typename"`
					} `json:"edges"`
					Typename string `json:"__typename"`
				} `json:"productOptions"`
				Metafields struct {
					Edges []struct {
						Node struct {
							ID       string `json:"id"`
							Key      string `json:"key"`
							Value    string `json:"value"`
							Typename string `json:"__typename"`
						} `json:"node"`
						Typename string `json:"__typename"`
					} `json:"edges"`
					Typename string `json:"__typename"`
				} `json:"metafields"`
				Typename string `json:"__typename"`
			} `json:"node"`
			Typename string `json:"__typename"`
		} `json:"edges"`
		Typename string `json:"__typename"`
	} `json:"variants"`
	Typename string `json:"__typename"`
}
