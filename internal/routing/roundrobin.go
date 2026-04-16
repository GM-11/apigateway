package routing

import (
	"sync"

	"example.com/m/v2/internal/utils"
)

type Counter struct {
	index int
	mu    sync.Mutex
}

type RoundRobin struct {
	counters map[string]*Counter
}

func NewRoundRobin(routes []utils.Route) *RoundRobin {
	rr := &RoundRobin{
		counters: make(map[string]*Counter),
	}
	rr.InitRR(routes)
	return rr
}

func (rr *RoundRobin) InitRR(routes []utils.Route) {
	counters := make(map[string]*Counter)
	for _, route := range routes {
		c := Counter{
			index: 0,
			mu:    sync.Mutex{},
		}

		counters[route.Prefix] = &c
	}
	rr.counters = counters
}

func (rr *RoundRobin) GetServerIndex(route utils.Route) int {
	r := rr.counters[route.Prefix]
	r.mu.Lock()
	defer r.mu.Unlock()

	val := r.index
	r.index = (r.index + 1) % len(route.Upstreams)
	return val
}
