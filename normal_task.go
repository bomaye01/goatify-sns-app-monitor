package main

import (
	"fmt"
	"time"
)

type NormalTask struct {
	*SnsTask
}

func NewNormalTask(taskName string) (*NormalTask, error) {
	normalTask := &NormalTask{
		SnsTask: &SnsTask{},
	}

	runCallback := func() {
		normalTask.loopMonitor()
	}
	stopCallback := func() {}

	baseTask, err := NewBaseTask(taskName, runCallback, stopCallback, proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task: %v", err)
	}

	normalTask.SnsTask.BaseTask = baseTask

	return normalTask, nil
}

func (t *NormalTask) loopMonitor() {
	configMu.RLock()
	defer time.Sleep(time.Millisecond * time.Duration(config.NormalTask.Timeout))
	configMu.RUnlock()

	t.rotateProxy()

	t.getProductsBySku()

	// productData := &ProductData{}

	// Compare new data with known product states
	// stateChanged := t.matchProductStates(productData)
	// if stateChanged {
	// 	go writeProductStates()
	// }
}
