package main

type NewArrivalsResponse struct {
	Response *Response `json:"response"`
}

type Response struct {
	ProductNodes []*ProductNodeReference `json:"docs"`
}

type ProductNodeReference struct {
	ProductType string `json:"product_type"`
	Sku         string `json:"sku"`
	Pid         string `json:"pid"`
}

type ProductsBySkusResponse struct {
	Data *Data `json:"data"`
}

type Data struct {
	Site *Site `json:"site"`
}

type Site struct {
	Search *Search `json:"search"`
}

type Search struct {
	SearchProducts *SearchProducts `json:"searchProducts"`
}

type SearchProducts struct {
	Products []*ProductReference `json:"products"`
}

type ProductReference struct {
	Edges []*ProductEdge `json:"edges"`
}

type ProductEdge struct {
	Nodes *ProductNode `json:"node"`
}

type ProductNode struct {
	Sku          string         `json:"sku"`
	Name         string         `json:"name"`
	Prices       *Prices        `json:"prices"`
	DefaultImage *DefaultImage  `json:"defaultImage"`
	CustomFields []*CustomField `json:"customFields"`
	MetaFields   []*MetaField   `json:"metafields"`
	Variants     []*Variant     `json:"variants"`
}

type Prices struct {
	Price *Price `json:"price"`
}

type Price struct {
	CurrencyCode string `json:"currencyCode"`
	Value        int    `json:"value"`
}

type DefaultImage struct {
	Url string `json:"url"`
}

type CustomField struct {
	Edges []*CustomFieldEdge `json:"edges"`
}

type CustomFieldEdge struct {
	Node *CustomFieldNode `json:"node"`
}

type CustomFieldNode struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type MetaField struct {
	Edges []*MetaFieldEdge `json:"edges"`
}

type MetaFieldEdge struct {
	Node *MetaFieldNode `json:"node"`
}

type MetaFieldNode struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Variant struct {
	Edges []*VariantEdge `json:"edges"`
}

type VariantEdge struct {
	Node *VariantNode `json:"node"`
}

type VariantNode struct {
	MetaFields []*MetaField `json:"metafields"`
	Inventory  *Inventory   `json:"inventory"`
}

type Inventory struct {
	IsInStock bool `json:"isInStock"`
}
