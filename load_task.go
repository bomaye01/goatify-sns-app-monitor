package main

import (
	"fmt"
	"strings"
	"time"
)

type KwdQuery struct {
	rawQueryStr       string
	inclusiveKeywords []string
	exclusiveKeywords []string
	orKeywordsGroups  [][]string
}
type SkuQuery string

type QueryType int

const (
	SkuQueryType     QueryType = iota
	KeywordQueryType QueryType = iota
)

type LoadTask struct {
	*SnsTask
}

func NewLoadTask(taskName string) (*LoadTask, error) {
	loadTask := &LoadTask{
		SnsTask: &SnsTask{},
	}

	runCallback := func() {
		loadTask.loopMonitor()
	}
	stopCallback := func() {}

	baseTask, err := NewBaseTask(taskName, runCallback, stopCallback, proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task: %v", err)
	}

	loadTask.BaseTask = baseTask

	return loadTask, nil
}

func (t *LoadTask) addSkuQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	skuQuery := SkuQuery(strings.ToUpper(query))

	t.skuQueries = append(t.skuQueries, skuQuery)
}

func (t *LoadTask) addKeywordQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	query = strings.ToLower(query)
	q := createKeywordQuery(query)

	t.kwdQueries = append(t.kwdQueries, q)
}

func (t *LoadTask) removeSkuQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	skuQuery := SkuQuery(strings.ToUpper(query))

	for i, q := range t.skuQueries {
		if q == skuQuery {
			t.skuQueries = append(t.skuQueries[:i], t.skuQueries[i+1:]...)
			break
		}
	}
}

func (t *LoadTask) removeKeywordQuery(query string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	query = strings.ToLower(query)
	q := createKeywordQuery(query)

	for i, kwdQuery := range t.kwdQueries {
		if kwdQuery.rawQueryStr == q.rawQueryStr {
			t.kwdQueries = append(t.kwdQueries[:i], t.kwdQueries[i+1:]...)
			break
		}
	}
}

func (t *LoadTask) loopMonitor() {
	configMu.RLock()
	defer time.Sleep(time.Millisecond * time.Duration(config.LoadTask.Timeout))
	configMu.RUnlock()

	t.rotateProxy()

}
