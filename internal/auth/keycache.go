package auth

import (
	"crypto/rsa"
	"fmt"
	"sync"
	"time"
)

type KeyCache struct {
	keys        map[string]*rsa.PublicKey
	mu          sync.RWMutex
	lastFetched time.Time
	ttl         time.Duration
}

func NewKeyCache(ttl time.Duration) *KeyCache {
	return &KeyCache{
		keys:        make(map[string]*rsa.PublicKey),
		ttl:         ttl,
		mu:          sync.RWMutex{},
		lastFetched: time.Time{}}

}

func (kc *KeyCache) GetKey(kid string) (*rsa.PublicKey, error) {

	kc.mu.RLock()
	if time.Since(kc.lastFetched) < kc.ttl {
		key, exists := kc.keys[kid]
		if exists {
			kc.mu.RUnlock()
			return key, nil
		}
	}
	kc.mu.RUnlock()
	kc.mu.Lock()
	defer kc.mu.Unlock()

	// double check after acquiring write lock
	key, exists := kc.keys[kid]

	if exists {
		return key, nil
	}
	pubkeys, err := FetchJWKS()
	if err != nil {
		return nil, err
	}
	key, exists = pubkeys[kid]
	if exists {
		kc.keys = pubkeys
		kc.lastFetched = time.Now()
		return key, nil
	} else {
		return nil, fmt.Errorf("Kid not found %s", kid)
	}
}
