package wal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempWAL(t *testing.T) *FileWAL {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.wal")
	w, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { w.Close() })
	return w
}

func TestWAL_AppendAndReplay(t *testing.T) {
	w := tempWAL(t)
	ctx := context.Background()

	records := []Record{
		{Op: OpSet, Key: "key1", Value: []byte("val1"), ExpiresAt: time.Now().Add(time.Hour)},
		{Op: OpSet, Key: "key2", Value: []byte("val2")},
		{Op: OpDelete, Key: "key1"},
	}

	for _, r := range records {
		if err := w.Append(ctx, r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	var replayed []Record
	err := w.Replay(ctx, func(r Record) error {
		replayed = append(replayed, r)
		return nil
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if len(replayed) != len(records) {
		t.Fatalf("replayed %d records, want %d", len(replayed), len(records))
	}

	for i, r := range replayed {
		if r.Op != records[i].Op {
			t.Errorf("record %d: op = %d, want %d", i, r.Op, records[i].Op)
		}
		if r.Key != records[i].Key {
			t.Errorf("record %d: key = %q, want %q", i, r.Key, records[i].Key)
		}
		if string(r.Value) != string(records[i].Value) {
			t.Errorf("record %d: value = %q, want %q", i, r.Value, records[i].Value)
		}
	}
}

func TestWAL_Truncate(t *testing.T) {
	w := tempWAL(t)
	ctx := context.Background()

	w.Append(ctx, Record{Op: OpSet, Key: "k", Value: []byte("v")})

	if err := w.Truncate(ctx); err != nil {
		t.Fatalf("Truncate: %v", err)
	}

	var count int
	w.Replay(ctx, func(r Record) error {
		count++
		return nil
	})
	if count != 0 {
		t.Errorf("got %d records after truncate, want 0", count)
	}
}

func TestWAL_CorruptRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.wal")

	w, _ := Open(path)
	ctx := context.Background()
	w.Append(ctx, Record{Op: OpSet, Key: "good", Value: []byte("data")})
	w.Close()

	// Corrupt the last byte (CRC)
	data, _ := os.ReadFile(path)
	data[len(data)-1] ^= 0xFF
	os.WriteFile(path, data, 0644)

	w2, _ := Open(path)
	defer w2.Close()

	var count int
	w2.Replay(ctx, func(r Record) error {
		count++
		return nil
	})
	// Corrupt CRC should stop replay
	if count != 0 {
		t.Errorf("got %d records from corrupt WAL, want 0", count)
	}
}

func TestWAL_EmptyReplay(t *testing.T) {
	w := tempWAL(t)
	ctx := context.Background()

	var count int
	err := w.Replay(ctx, func(r Record) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("Replay on empty WAL: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d records, want 0", count)
	}
}

func TestWAL_LargeValue(t *testing.T) {
	w := tempWAL(t)
	ctx := context.Background()

	largeVal := make([]byte, 1024*1024) // 1MB
	for i := range largeVal {
		largeVal[i] = byte(i % 256)
	}

	w.Append(ctx, Record{Op: OpSet, Key: "big", Value: largeVal})

	var got []byte
	w.Replay(ctx, func(r Record) error {
		got = r.Value
		return nil
	})
	if len(got) != len(largeVal) {
		t.Fatalf("replayed value len = %d, want %d", len(got), len(largeVal))
	}
	for i := range got {
		if got[i] != largeVal[i] {
			t.Fatalf("byte %d differs", i)
		}
	}
}
