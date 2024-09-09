package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type Status string
type key int

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusReady   Status = "ready"
)

const (
	statusKey key = iota
)

type BaseTask struct {
	mu             sync.Mutex
	ctx            context.Context
	cancelCtx      context.CancelFunc
	runCallback    func()
	stopCallback   func()
	taskName       string
	logger         *Logger
	httpClient     tls_client.HttpClient
	proxyHandler   *ProxyHandler
	webhookHandler *WebhookHandler
	proxy          *proxy
}

func NewBaseTask(taskName string, runCallback func(), stopCallback func(), proxyHandler *ProxyHandler, webhookHandler *WebhookHandler) (*BaseTask, error) {
	if proxyHandler == nil {
		return nil, errors.New("proxy handler reference nil")
	}
	if webhookHandler == nil {
		return nil, errors.New("webhook handler reference nil")
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(10),
		tls_client.WithClientProfile(profiles.Okhttp4Android13),
		tls_client.WithNotFollowRedirects(),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("error creating http client: %v", err)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, statusKey, StatusReady)

	return &BaseTask{
		mu:             sync.Mutex{},
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		runCallback:    runCallback,
		stopCallback:   stopCallback,
		taskName:       taskName,
		logger:         NewLogger(taskName),
		httpClient:     client,
		proxyHandler:   proxyHandler,
		webhookHandler: webhookHandler,
		proxy:          nil,
	}, nil
}

func (b *BaseTask) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ctx.Value(statusKey).(Status) != StatusReady {
		return errors.New("cannot start task: not ready")
	}

	b.updateStatus(StatusRunning)

	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			default:
			}

			b.runCallback()
		}
	}()

	return nil
}

func (b *BaseTask) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cancelCtx()
	b.updateStatus(StatusStopped)

	b.stopCallback()
}

func (b *BaseTask) Recover() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.updateStatus(StatusReady)
}

func (b *BaseTask) GetStatus() Status {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.ctx.Value(statusKey).(Status)
}

func (b *BaseTask) WaitForTermination() {
	<-b.ctx.Done()
}

func (b *BaseTask) updateStatus(status Status) {
	b.ctx = context.WithValue(b.ctx, statusKey, status)
}

func (b *BaseTask) rotateProxy() {
	b.proxyHandler.ReleaseProxy(b.proxy)

	b.mu.Lock()
	b.proxy = b.proxyHandler.GetProxy()
	b.mu.Unlock()

	if b.proxy != nil {
		proxyStr := ProxyAsString(*b.proxy)

		err := b.httpClient.SetProxy(proxyStr)
		if err != nil {
			b.logger.Red(fmt.Sprintf("error setting proxy: %v", err))
		}
	} else {
		err := b.httpClient.SetProxy("")
		if err != nil {
			b.logger.Red(fmt.Sprintf("error setting proxy: %v", err))
		}
	}
}
