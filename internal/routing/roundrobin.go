package routing

import "sync"

type Counter struct {
	index int
	mu    sync.RWMutex
}

type RoundRobin struct {
	counters map[*Route]*Counter
}

func (rr *RoundRobin) InitRR(routes []Route) {
	counters := make(map[*Route]*Counter)
	for i := range routes {
		c := Counter{
			index: 0,
			mu:    sync.RWMutex{},
		}

		counters[&routes[i]] = &c
	}
	rr.counters = counters
}

func (rr *RoundRobin) GetServerIndex(route Route) int {
	r := rr.counters[&route]
	r.mu.Lock()
	defer r.mu.Unlock()

	val := r.index
	r.index = (r.index + 1) % len(route.Upstreams)
	return val
}
