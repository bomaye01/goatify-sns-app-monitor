package main

import (
	"fmt"
	"math/rand"
	"sync"
)

type proxy struct {
	host     string
	port     string
	username string
	password string
}

type ProxyHandler struct {
	mu            sync.Mutex
	logger        *Logger
	proxies       []*proxy
	proxyUsage    map[*proxy]int
	cond          *sync.Cond
	proxyfileName string
}

func NewProxyHandler(proxies []*proxy) *ProxyHandler {
	configMu.RLock()

	handler := &ProxyHandler{
		logger:        NewLogger("PROXY"),
		proxies:       proxies,
		proxyUsage:    make(map[*proxy]int),
		proxyfileName: config.ProxyfileName,
	}

	configMu.RUnlock()

	handler.cond = sync.NewCond(&handler.mu)

	if len(proxies) == 0 {
		handler.logger.Yellow("Warn: Running without proxies")
	} else {
		handler.shuffleProxies()
	}

	return handler
}

func (h *ProxyHandler) GetProxy() *proxy {
	configMu.RLock()
	defer configMu.RUnlock()
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for new proxyfile
	if config.ProxyfileName != h.proxyfileName {
		h.updateProxies()
	}

	if len(h.proxies) == 0 {
		return nil
	}

	for {
		for _, p := range h.proxies {
			if h.proxyUsage[p] < config.MaxTasksPerProxy {
				h.proxyUsage[p]++

				// Append proxy to end of slice to reduce its priority
				if len(h.proxies) > 1 {
					h.proxies = append(h.proxies, p)
					h.proxies = (h.proxies)[1:]
				}

				return p
			}
		}
		// If no proxy is available, wait until one becomes free
		h.cond.Wait()
	}
}

func (h *ProxyHandler) ReleaseProxy(p *proxy) {
	if p == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.proxyUsage[p] > 0 {
		h.proxyUsage[p]--
	}
	// Signal waiting tasks that a proxy might be available
	h.cond.Signal()
}

func (h *ProxyHandler) ReportBadProxy(p *proxy) {
	if p == nil {
		return
	}

	configMu.RLock()
	defer configMu.RUnlock()
	h.mu.Lock()
	defer h.mu.Unlock()

	if !config.RemoveBadProxy {
		return
	}

	// Search proxy index
	proxyIndex := -1
	for i, proxy := range h.proxies {
		if proxy == p {
			proxyIndex = i
			break
		}
	}

	// Check if found
	if proxyIndex >= 0 {
		// Remove bad proxy
		h.proxies = append(h.proxies[:proxyIndex], h.proxies[proxyIndex+1:]...)
		delete(h.proxyUsage, p)

		if len(h.proxies) == 0 {
			h.logger.Yellow("Warning: No good proxies left. Running without proxies")
		}

		writeProxyfile(h.proxyfileName, h.proxies)
	}
}

// Take locks before calling updateProxies! [configMu RLock, h.mu Lock]
func (h *ProxyHandler) updateProxies() {
	filenameOld := h.proxyfileName
	filenameNew := config.ProxyfileName
	if filenameOld == "" {
		filenameOld = "localhost"
	}
	if filenameNew == "" {
		filenameNew = "localhost"
	}

	h.logger.Yellow(fmt.Sprintf("Reloading proxyfile (%s -> %s)", filenameOld, filenameNew))

	h.proxyfileName = config.ProxyfileName

	newProxies, err := readProxyfile(h.proxyfileName)
	if err != nil {
		h.logger.Red(fmt.Sprintf("Reload proxyfile: %v (Sticking to old proxy list)", err))
		return
	}

	// Update the proxy list
	h.proxies = newProxies
	h.proxyfileName = config.ProxyfileName
	h.proxyUsage = make(map[*proxy]int)

	h.shuffleProxies()

	// Wake up all tasks so they can get new proxies
	h.cond.Broadcast()
}

// Take lock before calling updateProxies! [h.mu]
func (h *ProxyHandler) shuffleProxies() {
	rand.Shuffle(len(h.proxies), func(i, j int) { h.proxies[i], h.proxies[j] = h.proxies[j], h.proxies[i] })
}

func ProxyAsString(proxy proxy) string {
	if proxy.host != "" && proxy.port != "" {
		return fmt.Sprintf("http://%s:%s@%s:%s", proxy.username, proxy.password, proxy.host, proxy.port)
	}
	return ""
}
