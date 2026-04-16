package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"example.com/m/v2/internal/auth"
	"example.com/m/v2/internal/utils"
)

const MAX_TOKENS float64 = 1000.0

type RateLimiter struct {
	rwmu    sync.RWMutex
	buckets map[string]*TokenBucket
	configs map[string]*utils.RateLimit
}

func NewRateLimiter(routes []utils.Route) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		configs: make(map[string]*utils.RateLimit),
	}

	for _, route := range routes {
		if route.RateLimit != nil {
			rl.configs[route.Prefix] = route.RateLimit
		}
	}

	return rl
}

func (rl *RateLimiter) getBucket(key, prefix string) *TokenBucket {
	rl.rwmu.RLock()
	bucket, exists := rl.buckets[key]
	if exists {
		rl.rwmu.RUnlock()
		return bucket
	}
	rl.rwmu.RUnlock()
	rl.rwmu.Lock()
	defer rl.rwmu.Unlock()

	bucket, exists = rl.buckets[key]
	if exists {
		return bucket
	}

	config, _ := rl.configs[prefix]

	nb := &TokenBucket{lastRefill: time.Now(), tokens: config.Burst, maxTokens: config.Burst, refillRate: config.Rate}

	rl.buckets[key] = nb

	return nb
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userId, ok := auth.GetUserID(r)
		if !ok {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, "Unable to parse remote address", http.StatusInternalServerError)
				return
			}
			userId = host
		}

		if !ok {
			http.Error(w, "Route prefix does not exist", http.StatusNotFound)
			return
		}

		if _, exists := rl.configs[utils.GetRoutePrefixKey()]; !exists {
			next.ServeHTTP(w, r)
			return
		}

		tb := rl.getBucket(userId, utils.GetRoutePrefixKey())
		if !tb.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
