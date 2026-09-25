package safety

import (
	"errors"
	"sync"
)

var ErrRequestBudgetExceeded = errors.New("request budget exceeded")

type RequestBudget struct {
	mu        sync.Mutex
	max, used int
}
type RequestBudgetMetrics struct {
	MaxRequests  int `json:"max_requests"`
	RequestsUsed int `json:"requests_used"`
	Remaining    int `json:"remaining"`
}

func NewRequestBudget(max int) (*RequestBudget, error) {
	if max < 1 {
		return nil, errors.New("request budget max must be positive")
	}
	return &RequestBudget{max: max}, nil
}

// Acquire reserves exactly one outbound attempt. Call it immediately before
// handing a request to the network transport; failures and retries both count.
func (b *RequestBudget) Acquire() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.used >= b.max {
		return ErrRequestBudgetExceeded
	}
	b.used++
	return nil
}
func (b *RequestBudget) Metrics() RequestBudgetMetrics {
	b.mu.Lock()
	defer b.mu.Unlock()
	return RequestBudgetMetrics{b.max, b.used, b.max - b.used}
}
