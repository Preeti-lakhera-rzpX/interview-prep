package ratelimit

import (
	"sync"
	"testing"
)

func TestLimiter_AllowWithinCapacity(t *testing.T) {
	l := New(Config{Rate: 10, Capacity: 5})

	for i := 0; i < 5; i++ {
		if !l.Allow("user:1") {
			t.Errorf("call %d should be allowed within capacity", i+1)
		}
	}
}

func TestLimiter_DenyWhenExhausted(t *testing.T) {
	l := New(Config{Rate: 10, Capacity: 2})

	l.Allow("user:1")
	l.Allow("user:1")

	if l.Allow("user:1") {
		t.Error("third call should be denied (capacity=2)")
	}
}

func TestLimiter_DifferentKeysIndependent(t *testing.T) {
	l := New(Config{Rate: 10, Capacity: 1})

	l.Allow("user:1")

	if !l.Allow("user:2") {
		t.Error("different key should have its own bucket")
	}
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	l := New(Config{Rate: 1000, Capacity: 1000})

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			l.Allow("user:shared")
		}(i)
	}
	wg.Wait()
}
