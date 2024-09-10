package main

import (
	"fmt"
	"time"
)

type LoadTask struct {
	*SnsTask
	group *LoadTaskGroup
}

func NewLoadTask(taskName string, group *LoadTaskGroup) (*LoadTask, error) {
	if group == nil {
		return nil, fmt.Errorf("Error creating load task: group nil")
	}

	loadTask := &LoadTask{
		SnsTask: &SnsTask{},
		group:   group,
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

	res, err := t.getNewArrivals()
	if err != nil {
		t.logger.Red(err)
		return
	}

	go t.group.handleNewArrivalsResponse(res)

	t.rotateProxy()

}
