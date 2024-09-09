package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type NormalTaskGroup struct {
	*BaseTaskGroup
	skuQueryStrings []string
	normalTasks     []*NormalTask
}

func NewNormalTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler, skuQueryStrings []string) (*NormalTaskGroup, error) {
	normalTaskGroup := &NormalTaskGroup{
		skuQueryStrings: skuQueryStrings,
		normalTasks:     []*NormalTask{},
	}

	baseTaskGroup, err := NewBaseTaskGroup(proxyHandler, webhookHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating base task group: %v", err)
	}

	normalTaskGroup.BaseTaskGroup = baseTaskGroup

	return normalTaskGroup, nil
}

func (g *NormalTaskGroup) AddSkuQuery(skuStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuStr = strings.TrimSpace(strings.ToUpper(skuStr))

	g.skuQueryStrings = append(g.skuQueryStrings, skuStr)
}

func (g *NormalTaskGroup) RemoveSkuQuery(skuStr string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	skuStr = strings.TrimSpace(strings.ToUpper(skuStr))

	removeIndex := -1
	for i, query := range g.skuQueryStrings {
		if query == skuStr {
			removeIndex = i
			break
		}
	}

	if removeIndex >= 0 {
		g.skuQueryStrings = append(g.skuQueryStrings[:removeIndex], g.skuQueryStrings[removeIndex+1:]...)
	}
}

func (t *NormalTaskGroup) notifySize(productData *ProductData) {}

func (t *NormalTaskGroup) notifyPrice(productData *ProductData, oldPrice string, newPrice string) {}

func (g *NormalTaskGroup) AddTask(task *NormalTask) error {
	if task.GetStatus() != StatusReady {
		return &TaskNotReadyError{}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.normalTasks = append(g.normalTasks, task)

	return nil
}

func (g *NormalTaskGroup) RemoveTask(task *NormalTask) error {
	if task.GetStatus() == StatusRunning {
		return &TaskRunningError{}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	removeIndex := -1
	for i, t := range g.normalTasks {
		if t == task {
			removeIndex = i
			break
		}
	}
	if removeIndex >= 0 {
		g.normalTasks = append(g.normalTasks[:removeIndex], g.normalTasks[removeIndex+1:]...)
	}

	return nil
}

func (g *NormalTaskGroup) StartAllTasks() error {
	for _, task := range g.normalTasks {
		if task.GetStatus() != StatusReady {
			return &TaskNotReadyError{}
		}
	}

	for _, task := range g.normalTasks {
		tasksWg.Add(1)

		go func() {
			if config.NormalTask.BurstStart {
				offsetMilliseconds := rand.Intn(config.NormalTask.Timeout)
				time.Sleep(time.Millisecond * time.Duration(offsetMilliseconds))
			}
			task.Start()

			task.WaitForTermination()

			tasksWg.Done()
		}()
	}

	return nil
}

func (g *NormalTaskGroup) StopAllTasks() {
	for _, task := range g.normalTasks {
		task.Stop()
	}
}
