package main

import (
	"fmt"
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

func (t *LoadTask) loopMonitor() {
	configMu.RLock()
	defer time.Sleep(time.Millisecond * time.Duration(config.LoadTask.Timeout))
	configMu.RUnlock()

	t.rotateProxy()

}
