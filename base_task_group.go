package main

import (
	"errors"
	"sync"
)

type BaseTaskGroup struct {
	mu             sync.Mutex
	proxyHandler   *ProxyHandler
	webhookHandler *WebhookHandler
}

func NewBaseTaskGroup(proxyHandler *ProxyHandler, webhookHandler *WebhookHandler) (*BaseTaskGroup, error) {
	if proxyHandler == nil {
		return nil, errors.New("proxy handler reference nil")
	}
	if webhookHandler == nil {
		return nil, errors.New("webhook handler reference nil")
	}

	return &BaseTaskGroup{
		proxyHandler:   proxyHandler,
		webhookHandler: webhookHandler,
	}, nil
}
