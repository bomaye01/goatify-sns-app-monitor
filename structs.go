package main

// config.json
type Config struct {
	NormalTask          NormalTaskConfig `json:"normal"`
	LoadTask            LoadTaskConfig   `json:"load"`
	MaxTasksPerProxy    int              `json:"maxTasksPerProxy"`
	ProxyfileName       string           `json:"proxyfile"`
	WebhookErrorTimeout int              `json:"webhookErrorTimeoutInMilliseconds"`
	RemoveBadProxy      bool             `json:"autoRemoveBadProxy"`
}

type NormalTaskConfig struct {
	Timeout     int      `json:"timeoutInMilliseconds"`
	BurstStart  bool     `json:"burstStart"`
	WebhookUrls []string `json:"webhookUrls"`
	NumTasks    int      `json:"numTasks"`
}

type LoadTaskConfig struct {
	Timeout     int      `json:"timeoutInMilliseconds"`
	BurstStart  bool     `json:"burstStart"`
	WebhookUrls []string `json:"webhookUrls"`
	NumTasks    int      `json:"numTasks"`
}

// product_states.json
type ProductStates struct {
	Normal ProductStatesNormal `json:"normal"`
	Load   ProductStatesLoad   `json:"load"`
}

type ProductStatesNormal struct {
	ProductStates []*ProductStateNormal `json:"productStates"`
}

type ProductStateNormal struct {
	Sku            string   `json:"sku"`
	AvailableSizes []string `json:"availableSizes"`
	Price          string   `json:"price"`
}

type ProductStatesLoad struct {
	NotifiedProducts []*ProductStateLoad `json:"notifiedProducts"`
	LastKnownPid     string              `json:"lastKnownPid"`
	SkuQueries       []string            `json:"skuQueries"`
	KeywordQueries   []string            `json:"keywordQueries"`
}

type ProductStateLoad struct {
	Sku                    string   `json:"sku"`
	MatchedBySku           bool     `json:"matchedBySku"`
	MatchingKeywordQueries []string `json:"matchingKeywordQueries"`
}

type ProductData struct {
	ProductUrl     string
	Title          string
	Sku            string
	AvailableSizes []string
	Price          string
	ImageUrl       string
	IdentifyerStr  string
}

// Websocket receive structs
type MessageType struct {
	TypeName string `json:"typeName"`
}

type AddMessage struct {
	TypeName  string `json:"typeName"`
	TaskId    string `json:"taskId"`
	InputType string `json:"inputType"`
	AddQuery  string `json:"addQuery"`
}

type RemoveMessage struct {
	TypeName    string `json:"typeName"`
	TaskId      string `json:"taskId"`
	InputType   string `json:"inputType"`
	RemoveQuery string `json:"removeQuery"`
}

type ListMessage struct {
	TypeName  string `json:"typeName"`
	TaskId    string `json:"taskId"`
	InputType string `json:"inputType"`
}

// Websocket send structs

type SuccessResponse struct {
	TypeName    string `json:"typeName"`
	TaskId      string `json:"taskId"`
	SuccessTest string `json:"successText"`
}

type SuccessListResponse struct {
	TypeName    string   `json:"typeName"`
	TaskId      string   `json:"taskId"`
	SuccessTest string   `json:"successText"`
	List        []string `json:"list"`
}

type ErrorResponse struct {
	TypeName  string `json:"typeName"`
	TaskId    string `json:"taskId"`
	ErrorText string `json:"errorText"`
}
