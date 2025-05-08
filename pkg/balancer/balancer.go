package balancer

import (
	"LoadBalancer/internal/config"
	"context"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	healthCheckInterval = time.Minute * 5
	healthCheckPath     = "/ping"
)

type backendStatus struct {
	url    *url.URL
	active bool
}

type Balancer struct {
	mu       sync.Mutex
	backends []backendStatus
	idx      int
}

func NewBalancer(ctx context.Context, cfg config.Config) *Balancer {
	b := &Balancer{
		mu:       sync.Mutex{},
		backends: make([]backendStatus, 0),
	}
	for _, urlString := range cfg.Backends {
		parsedURL, err := url.Parse(urlString)
		if err != nil {
			log.Fatalf("Failed to parse backendStatus URL %s: %v", b, err)
		}
		b.backends = append(b.backends, backendStatus{
			url:    parsedURL,
			active: true,
		})
	}
	b.checkAllBackends()
	go b.StartHealthCheck(ctx)
	return b
}

func (b *Balancer) GetNextURL() *url.URL {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Проходим по кругу один раз в поисках активного бэкенда
	for range len(b.backends) {
		backend := b.backends[b.idx]
		if backend.active {
			b.idx++
			return backend.url
		}
		b.idx++
		if b.idx == len(b.backends) {
			b.idx = 0
		}
	}

	return nil
}

// StartHealthCheck выполняет переодическую проверку работоспособности серверов
func (b *Balancer) StartHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.checkAllBackends()
		case <-ctx.Done():
			return
		}
	}
}

// Проверка всех серверов
func (b *Balancer) checkAllBackends() {
	var wg sync.WaitGroup

	for i := range b.backends {
		wg.Add(1)
		go func(backendInst *backendStatus) {
			defer wg.Done()
			b.checkBackend(backendInst)
		}(&b.backends[i])
	}
	wg.Wait()
}

// Проверка одного сервера
func (b *Balancer) checkBackend(backend *backendStatus) {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckInterval)
	defer cancel()

	URL := backend.url.String() + healthCheckPath
	req, err := http.NewRequestWithContext(ctx, "GET", backend.url.String(), nil)
	if err != nil {
		b.markBackend(backend, false)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("server %s unavailable: %v", URL, err)
		b.markBackend(backend, false)
		return
	}
	defer resp.Body.Close()

	b.markBackend(backend, resp.StatusCode == 200)
}

// Обновление статуса сервера
func (b *Balancer) markBackend(backend *backendStatus, status bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	backend.active = status
}
