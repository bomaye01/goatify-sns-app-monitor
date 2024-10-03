package main

// config.json
type Config struct {
	NormalTask      NormalTaskConfig `json:"normal"`
	LoadTask        LoadTaskConfig   `json:"load"`
	DiscordPresence struct {
		AvatarUrl  string `json:"avatarUrl"`
		EmbedColor int    `json:"embedColor"`
		FooterText string `json:"footerText"`
	} `json:"discordPresence"`
	MaxTasksPerProxy    int    `json:"maxTasksPerProxy"`
	ProxyfileName       string `json:"proxyfile"`
	WebhookErrorTimeout int    `json:"webhookErrorTimeoutInMilliseconds"`
	RemoveBadProxy      bool   `json:"autoRemoveBadProxy"`
	InstanceName        string `json:"instanceName"`
	WebsocketPort       int    `json:"websocketPort"`
	EnableFileLogging   bool   `json:"enableFileLogging"`
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
	Sku              string          `json:"sku"`
	AvailableForSale bool            `json:"availableForSale"`
	AvailableSizes   []AvailableSize `json:"availableSizes"`
	Price            string          `json:"price"`
}

type ProductStatesLoad struct {
	NotifiedProducts []*ProductStateLoad `json:"notifiedProducts"`
	LastKnownPid     string              `json:"lastKnownPid"`
	KeywordQueries   []string            `json:"keywordQueries"`
}

type ProductStateLoad struct {
	Sku                    string   `json:"sku"`
	MatchingKeywordQueries []string `json:"matchingKeywordQueries"`
}

type ProductData struct {
	ProductUrl       string
	Title            string
	Sku              string
	AvailableForSale bool
	AvailableSizes   []AvailableSize
	Price            string
	ImageUrl         string
	IdentifyerStr    string
}

type AvailableSize struct {
	Name          string `json:"Name"`
	AmountInStock int    `json:"AmountInStock"`
}
