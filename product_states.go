package main

var productStates *ProductStates = nil

func NormalSetState(skuStr string, state *ProductStateNormal) {
	if _, i := NormalGetState(skuStr); i >= 0 {
		productStates.Normal.ProductStates[i] = state
	} else {
		productStates.Normal.ProductStates = append(productStates.Normal.ProductStates, state)
	}
}

func NormalUnsetState(skuStr string) error {
	_, i := NormalGetState(skuStr)
	if i == -1 {
		return &NotIncludedError{
			statesType:    "normal",
			includedType:  "sku",
			includedValue: skuStr,
		}
	}

	productStates.Normal.ProductStates = append(productStates.Normal.ProductStates[:i], productStates.Normal.ProductStates[i+1:]...)

	return nil
}

func NormalGetState(skuStr string) (*ProductStateNormal, int) {
	for i, state := range productStates.Normal.ProductStates {
		if state.Sku == skuStr {
			return state, i
		}
	}

	return nil, -1
}

func NormalGetAllSkus() []string {
	skus := []string{}

	for _, state := range productStates.Normal.ProductStates {
		skus = append(skus, state.Sku)
	}

	return skus
}

func LoadAddKwd(kwdStr string) error {
	if i := LoadGetIndexKwd(kwdStr); i >= 0 {
		return &AlreadyIncludedError{
			statesType:    "load",
			includedType:  "kwd",
			includedValue: kwdStr,
		}
	}

	productStates.Load.KeywordQueries = append(productStates.Load.KeywordQueries, kwdStr)

	return nil
}

func LoadRemoveKwd(kwdStr string) error {
	i := LoadGetIndexKwd(kwdStr)
	if i == -1 {
		return &NotIncludedError{
			statesType:    "load",
			includedType:  "kwd",
			includedValue: kwdStr,
		}
	}

	productStates.Load.KeywordQueries = append(productStates.Load.KeywordQueries[:i], productStates.Load.KeywordQueries[i+1:]...)

	return nil
}

func LoadGetIndexKwd(kwdStr string) int {
	for i, sku := range productStates.Load.KeywordQueries {
		if sku == kwdStr {
			return i
		}
	}

	return -1
}

func LoadSetLastKnownPid(pid string) {
	productStates.Load.LastKnownPid = pid
}

func LoadGetLastKnownPid(pid string) string {
	return productStates.Load.LastKnownPid
}
