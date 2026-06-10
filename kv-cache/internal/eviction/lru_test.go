package eviction

import "testing"

func TestLRU_Add(t *testing.T) {
	l := newLRU(3)

	tests := []struct {
		name    string
		key     string
		wantEvt string
		wantLen int
	}{
		{"first", "a", "", 1},
		{"second", "b", "", 2},
		{"third", "c", "", 3},
		{"overflow evicts oldest", "d", "a", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evicted := l.Add(tt.key)
			if evicted != tt.wantEvt {
				t.Errorf("Add(%q) evicted = %q, want %q", tt.key, evicted, tt.wantEvt)
			}
			if l.Len() != tt.wantLen {
				t.Errorf("Len() = %d, want %d", l.Len(), tt.wantLen)
			}
		})
	}
}

func TestLRU_AccessPromotes(t *testing.T) {
	l := newLRU(3)
	l.Add("a")
	l.Add("b")
	l.Add("c")

	// Access "a" to promote it; next eviction should remove "b"
	l.Access("a")
	evicted := l.Add("d")
	if evicted != "b" {
		t.Errorf("expected 'b' evicted after accessing 'a', got %q", evicted)
	}
}

func TestLRU_Remove(t *testing.T) {
	l := newLRU(3)
	l.Add("a")
	l.Add("b")
	l.Add("c")

	l.Remove("b")
	if l.Len() != 2 {
		t.Fatalf("Len() = %d after remove, want 2", l.Len())
	}

	// Adding should not evict since we're under capacity
	evicted := l.Add("d")
	if evicted != "" {
		t.Errorf("unexpected eviction %q after remove freed space", evicted)
	}
}

func TestLRU_AddDuplicate(t *testing.T) {
	l := newLRU(3)
	l.Add("a")
	evicted := l.Add("a")
	if evicted != "" {
		t.Errorf("duplicate Add should not evict, got %q", evicted)
	}
	if l.Len() != 1 {
		t.Errorf("Len() = %d after duplicate add, want 1", l.Len())
	}
}

func TestLRU_AccessNonexistent(t *testing.T) {
	l := newLRU(3)
	l.Access("nonexistent") // should not panic
}
