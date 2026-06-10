package eviction

import "testing"

func TestFIFO_EvictsOldest(t *testing.T) {
	f := newFIFO(3)
	f.Add("a")
	f.Add("b")
	f.Add("c")

	evicted := f.Add("d")
	if evicted != "a" {
		t.Errorf("expected 'a' evicted (FIFO), got %q", evicted)
	}
}

func TestFIFO_AccessDoesNotAffectOrder(t *testing.T) {
	f := newFIFO(3)
	f.Add("a")
	f.Add("b")
	f.Add("c")

	// Access "a" — should NOT prevent it from being evicted
	f.Access("a")
	evicted := f.Add("d")
	if evicted != "a" {
		t.Errorf("expected 'a' evicted despite access, got %q", evicted)
	}
}

func TestFIFO_Remove(t *testing.T) {
	f := newFIFO(3)
	f.Add("a")
	f.Add("b")
	f.Add("c")

	f.Remove("a")
	if f.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", f.Len())
	}

	// Should not evict since under capacity
	evicted := f.Add("d")
	if evicted != "" {
		t.Errorf("should not evict with space available, got %q", evicted)
	}

	// Now at capacity; next eviction should be "b" (oldest remaining)
	evicted = f.Add("e")
	if evicted != "b" {
		t.Errorf("expected 'b' evicted, got %q", evicted)
	}
}

func TestFIFO_AddDuplicate(t *testing.T) {
	f := newFIFO(3)
	f.Add("a")
	evicted := f.Add("a")
	if evicted != "" {
		t.Errorf("duplicate Add should not evict, got %q", evicted)
	}
	if f.Len() != 1 {
		t.Errorf("Len() = %d, want 1", f.Len())
	}
}

func TestFIFO_SequentialEviction(t *testing.T) {
	f := newFIFO(2)
	f.Add("a")
	f.Add("b")

	tests := []struct {
		add  string
		want string
	}{
		{"c", "a"},
		{"d", "b"},
		{"e", "c"},
	}

	for _, tt := range tests {
		evicted := f.Add(tt.add)
		if evicted != tt.want {
			t.Errorf("Add(%q) evicted = %q, want %q", tt.add, evicted, tt.want)
		}
	}
}
