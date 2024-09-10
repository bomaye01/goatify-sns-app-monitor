package main

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

type BaseTaskGroup struct {
	mu             sync.Mutex
	proxyHandler   *ProxyHandler
	webhookHandler *WebhookHandler
	logger         *Logger
	baseTasks      []*BaseTask
}

func NewBaseTaskGroup(taskName string, proxyHandler *ProxyHandler, webhookHandler *WebhookHandler) (*BaseTaskGroup, error) {
	if proxyHandler == nil {
		return nil, errors.New("proxy handler reference nil")
	}
	if webhookHandler == nil {
		return nil, errors.New("webhook handler reference nil")
	}

	return &BaseTaskGroup{
		proxyHandler:   proxyHandler,
		webhookHandler: webhookHandler,
		logger:         NewLogger(taskName),
	}, nil
}

func (g *BaseTaskGroup) AddTask(task *BaseTask) error {
	if task.GetStatus() != StatusReady {
		return &TaskNotReadyError{}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.baseTasks = append(g.baseTasks, task)

	return nil
}

func (g *BaseTaskGroup) RemoveTask(task *BaseTask) error {
	if task.GetStatus() == StatusRunning {
		return &TaskRunningError{}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	removeIndex := -1
	for i, t := range g.baseTasks {
		if t == task {
			removeIndex = i
			break
		}
	}
	if removeIndex >= 0 {
		g.baseTasks = append(g.baseTasks[:removeIndex], g.baseTasks[removeIndex+1:]...)
	}

	return nil
}

func (g *BaseTaskGroup) StartAllTasks() error {
	for _, task := range g.baseTasks {
		if task.GetStatus() != StatusReady {
			return &TaskNotReadyError{}
		}
	}

	for _, task := range g.baseTasks {
		tasksWg.Add(1)

		go func() {
			if config.NormalTask.BurstStart {
				offsetMilliseconds := rand.Intn(config.NormalTask.Timeout)
				time.Sleep(time.Millisecond * time.Duration(offsetMilliseconds))
			}
			task.start()

			task.WaitForTermination()

			tasksWg.Done()
		}()
	}

	return nil
}

func (g *BaseTaskGroup) StopAllTasks() {
	for _, task := range g.baseTasks {
		task.stop()
	}
}
