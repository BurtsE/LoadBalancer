package rate_limiter

import (
	"context"
	"sync"
	"time"
)

// tokenBucket хранит токены. При создании запускает горутину,
// по таймеру пополняющую число токенов. При удалении бакетов необходимо вызвать
// метод clear для уничтожения горутины
type tokenBucket struct {
	capacity   int                // Макс. токенов
	refillRate int                // Таймер пополнения токенов в минуту
	tokens     int                // Текущее количество
	mu         sync.Mutex         // Блокировка для конкурентности
	cancelFunc context.CancelFunc // Функция для остановки горутины, заполняющей бакет
}

func newTokenBucket(capacity, refillRate int) *tokenBucket {
	ctx, cancel := context.WithCancel(context.Background())
	b := &tokenBucket{
		capacity:   capacity,
		refillRate: refillRate,
		tokens:     capacity,
		cancelFunc: cancel,
	}

	go b.refill(ctx)

	return b
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *tokenBucket) refill(ctx context.Context) {
	t := time.NewTicker(time.Duration(tb.refillRate) * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tb.mu.Lock()
			tb.tokens = min(tb.tokens+1, tb.capacity)
			tb.mu.Unlock()
		}
	}
}
func (tb *tokenBucket) clear() {
	tb.cancelFunc()
}
