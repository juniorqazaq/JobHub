package safety

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRequestBudgetExactBoundary(t *testing.T) {
	b, _ := NewRequestBudget(3)
	for i := 0; i < 3; i++ {
		if err := b.Acquire(); err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(b.Acquire(), ErrRequestBudgetExceeded) {
		t.Fatal("request four was not blocked")
	}
	m := b.Metrics()
	if m.MaxRequests != 3 || m.RequestsUsed != 3 || m.Remaining != 0 {
		t.Fatalf("%+v", m)
	}
}
func TestFailedAttemptsAndRetriesConsumeBudget(t *testing.T) {
	b, _ := NewRequestBudget(3)
	network := 0
	attempt := func(fail bool) error {
		if err := b.Acquire(); err != nil {
			return err
		}
		network++
		if fail {
			return errors.New("network failed")
		}
		return nil
	}
	_ = attempt(true)
	_ = attempt(true)
	_ = attempt(false)
	if !errors.Is(attempt(false), ErrRequestBudgetExceeded) || network != 3 {
		t.Fatalf("network=%d metrics=%+v", network, b.Metrics())
	}
}
func TestRequestBudgetConcurrent(t *testing.T) {
	b, _ := NewRequestBudget(3)
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Acquire() == nil {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 3 || b.Metrics().RequestsUsed != 3 {
		t.Fatalf("allowed=%d metrics=%+v", allowed.Load(), b.Metrics())
	}
}
