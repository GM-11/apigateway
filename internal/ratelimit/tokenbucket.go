package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	refillRate float64
	maxTokens  float64
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	timeElapsed := time.Since(tb.lastRefill)
	tb.tokens += timeElapsed.Seconds() * tb.refillRate

	tb.tokens = min(tb.tokens, tb.maxTokens)

	tb.lastRefill = time.Now()
	if tb.tokens >= 1 {
		tb.tokens -= 1
		return true
	}
	return false
}
