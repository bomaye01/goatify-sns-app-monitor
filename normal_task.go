package main

import (
	"fmt"
	"time"
)

type NormalTask struct {
	*SnsTask
	group *NormalTaskGroup
}

func NewNormalTask(taskName string, group *NormalTaskGroup) (*NormalTask, error) {
	if group == nil {
		return nil, fmt.Errorf("Error creating normal task: group nil")
	}

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

	res, err := t.getProductsBySku()
	if err != nil {
		t.logger.Red(err)
		return
	}

	go t.group.checkProductsBySkusResponse(res)
}
