package eviction

import "testing"

func TestLFU_EvictsLeastFrequent(t *testing.T) {
	l := newLFU(3)
	l.Add("a")
	l.Add("b")
	l.Add("c")

	// Access "a" and "c" so "b" has the lowest frequency
	l.Access("a")
	l.Access("c")

	evicted := l.Add("d")
	if evicted != "b" {
		t.Errorf("expected 'b' (least frequent) evicted, got %q", evicted)
	}
}

func TestLFU_TieBreaksOldest(t *testing.T) {
	l := newLFU(3)
	l.Add("a") // freq 1
	l.Add("b") // freq 1
	l.Add("c") // freq 1

	// All have freq 1, oldest ("a") should be evicted
	evicted := l.Add("d")
	if evicted != "a" {
		t.Errorf("expected 'a' (oldest with same freq) evicted, got %q", evicted)
	}
}

func TestLFU_FrequencyTracking(t *testing.T) {
	l := newLFU(3)
	l.Add("a")
	l.Add("b")
	l.Add("c")

	// Bump "a" to freq 3, "b" to freq 2, "c" stays at freq 1
	l.Access("a")
	l.Access("a")
	l.Access("b")

	// Should evict "c" (freq 1)
	evicted := l.Add("d")
	if evicted != "c" {
		t.Errorf("expected 'c' evicted, got %q", evicted)
	}

	// Now "b"(freq 2) < "a"(freq 3), "d"(freq 1) is lowest
	evicted = l.Add("e")
	if evicted != "d" {
		t.Errorf("expected 'd' evicted, got %q", evicted)
	}
}

func TestLFU_Remove(t *testing.T) {
	l := newLFU(3)
	l.Add("a")
	l.Add("b")
	l.Add("c")

	l.Remove("b")
	if l.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", l.Len())
	}

	evicted := l.Add("d")
	if evicted != "" {
		t.Errorf("should not evict with space available, got %q", evicted)
	}
}

func TestLFU_AddDuplicate(t *testing.T) {
	l := newLFU(3)
	l.Add("a")
	evicted := l.Add("a")
	if evicted != "" {
		t.Errorf("duplicate Add should not evict, got %q", evicted)
	}
	if l.Len() != 1 {
		t.Errorf("Len() = %d, want 1", l.Len())
	}
}
