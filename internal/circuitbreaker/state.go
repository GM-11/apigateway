package circuitbreaker

import (
	"sync"
	"time"
)

const (
	StateClosed = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	currentState           int
	failureCount           int
	circuitOpenedTimestamp time.Time
	firstFailureTimestamp  time.Time
	mu                     sync.Mutex
}

func (cb *CircuitBreaker) Allow(recoveryWindow time.Duration) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.currentState {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.circuitOpenedTimestamp) > recoveryWindow {
			cb.currentState = StateHalfOpen
			return true
		}

	case StateHalfOpen:
		cb.currentState = StateOpen
	default:
		return false
	}

	return false
}

func (cb *CircuitBreaker) RecordFailure(failureThreshold int, failureWindow time.Duration) {

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.currentState {
	case StateClosed:
		if time.Since(cb.firstFailureTimestamp) > failureWindow {
			cb.failureCount = 0
			cb.firstFailureTimestamp = time.Time{}
		}
		if cb.failureCount == 0 {
			cb.firstFailureTimestamp = time.Now()
		}
		cb.failureCount++
		if cb.failureCount > failureThreshold {
			cb.currentState = StateOpen
			cb.circuitOpenedTimestamp = time.Now()
		}
	case StateHalfOpen:
		cb.currentState = StateOpen
		cb.circuitOpenedTimestamp = time.Now()
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.currentState == StateHalfOpen {
		cb.currentState = StateClosed
		cb.failureCount = 0
		cb.firstFailureTimestamp = time.Time{}
		cb.circuitOpenedTimestamp = time.Time{}
	}
}
