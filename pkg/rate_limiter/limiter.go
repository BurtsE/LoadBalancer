package rate_limiter

import "sync"

// RateLimiter содержит таблицу бакетов, стандартные значения для бакетов
// и RWMutex для конкурентного доступа к таблице
type RateLimiter struct {
	buckets    map[string]*tokenBucket
	defaultCap int
	defaultRef int
	mu         sync.RWMutex
}

func NewRateLimiter(defaultCap, defaultRef int) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		defaultCap: defaultCap,
		defaultRef: defaultRef,
	}
}

// Allow проверяет лимит для клиента.
// Создает бакеты для новых клиентов
func (r *RateLimiter) Allow(clientID string) bool {
	r.mu.RLock()
	bucket, exists := r.buckets[clientID]
	r.mu.RUnlock()

	if !exists {
		r.mu.Lock()
		bucket = newTokenBucket(r.defaultCap, r.defaultRef)
		r.buckets[clientID] = bucket
		r.mu.Unlock()
	}

	return bucket.allow()
}

// SetCustomLimit устанавливает лимиты для отдельного пользователя
func (r *RateLimiter) SetCustomLimit(clientID string, capacity, refillRate int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	bucket, exists := r.buckets[clientID]
	if exists {
		bucket.clear()
	}
	r.buckets[clientID] = newTokenBucket(capacity, refillRate)
}

// Clear очищает бакеты
func (r *RateLimiter) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, bucket := range r.buckets {
		bucket.clear()
		delete(r.buckets, id)
	}
}
