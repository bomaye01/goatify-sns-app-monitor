package main

var defaultConfig Config = Config{
	NormalTask: NormalTaskConfig{
		Timeout:     5000,
		BurstStart:  true,
		WebhookUrls: []string{},
	},
	LoadTask: LoadTaskConfig{
		Timeout:     5000,
		BurstStart:  true,
		WebhookUrls: []string{},
	},
	MaxTasksPerProxy:    2,
	ProxyfileName:       "",
	WebhookErrorTimeout: 3500,
	RemoveBadProxy:      false,
}

var defaultProductStates ProductStates = ProductStates{
	Normal: ProductStatesNormal{
		SKUs:          []string{},
		ProductStates: []*ProductStateNormal{},
	},
	Load: ProductStatesLoad{
		NotifiedProducts: []*ProductStateLoad{},
		LastKnownPid:     "",
		SkuQueries:       []string{},
		KeywordQueries:   []string{},
	},
}
