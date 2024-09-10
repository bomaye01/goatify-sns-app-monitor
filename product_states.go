package main

import "fmt"

var productStates *ProductStates = nil

type AlreadyIncludedError struct {
	statesType    string
	includedType  string
	includedValue string
}

func (e *AlreadyIncludedError) Error() string {
	return fmt.Sprintf("%s \"%s\" already included in %s product states", e.includedType, e.includedValue, e.statesType)
}

type NotIncludedError struct {
	statesType    string
	includedType  string
	includedValue string
}

func (e *NotIncludedError) Error() string {
	return fmt.Sprintf("%s \"%s\" not included in %s product states", e.includedType, e.includedValue, e.statesType)
}

func NormalAddState(skuStr string, productData ProductData) error {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	if _, i := normalGetStateNoMu(skuStr); i >= 0 {
		return &AlreadyIncludedError{
			statesType:    "normal",
			includedType:  "sku",
			includedValue: skuStr,
		}
	}

	addStates := &ProductStateNormal{
		Sku:            productData.Sku,
		AvailableSizes: productData.AvailableSizes,
		Price:          productData.Price,
	}

	productStates.Normal.ProductStates = append(productStates.Normal.ProductStates, addStates)

	go writeProductStates()

	return nil
}

func NormalRemoveState(skuStr string) error {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	_, i := normalGetStateNoMu(skuStr)
	if i == -1 {
		return &NotIncludedError{
			statesType:    "normal",
			includedType:  "sku",
			includedValue: skuStr,
		}
	}

	productStates.Normal.ProductStates = append(productStates.Normal.ProductStates[:i], defaultProductStates.Normal.ProductStates[i+1:]...)

	go writeProductStates()

	return nil
}

func NormalGetState(skuStr string) (*ProductStateNormal, int) {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	return normalGetStateNoMu(skuStr)
}

func normalGetStateNoMu(skuStr string) (*ProductStateNormal, int) {
	for i, state := range productStates.Normal.ProductStates {
		if state.Sku == skuStr {
			return state, i
		}
	}

	return nil, -1
}

func NormalGetAllSkus() []string {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()

	skus := []string{}

	for _, state := range productStates.Normal.ProductStates {
		skus = append(skus, state.Sku)
	}

	return skus
}

func LoadAddSku(skuStr string) error {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	if i := loadGetIndexSkuNoMu(skuStr); i >= 0 {
		return &AlreadyIncludedError{
			statesType:    "load",
			includedType:  "sku",
			includedValue: skuStr,
		}
	}

	productStates.Load.SkuQueries = append(productStates.Load.SkuQueries, skuStr)

	go writeProductStates()

	return nil
}

func LoadRemoveSku(skuStr string) {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()
}

func LoadGetIndexSku(skuStr string) int {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	return loadGetIndexSkuNoMu(skuStr)
}

func loadGetIndexSkuNoMu(skuStr string) int {
	return -1 // Index
}

func LoadAddKwd(kwdStr string) {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()
}

func LoadRemoveKwd(kwdStr string) {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()
}

func LoadGetIndexKwd(kwdStr string) int {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	return loadGetIndexKwdNoMu(kwdStr)
}

func loadGetIndexKwdNoMu(kwdStr string) int {
	return -1 // Index
}

func LoadSetLastKnownPid(pid string) {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()
}

func LoadGetLastKnownPid(pid string) {
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()
}
