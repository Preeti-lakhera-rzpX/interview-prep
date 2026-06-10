package dedup

import (
	"sync"
	"testing"
	"time"

	"interview-prep/internal/model"
)

func TestDeduplicator_FirstCallNotDuplicate(t *testing.T) {
	d := New(1 * time.Minute)
	defer d.Close()

	payload := model.Payload{To: "a@b.com", Body: "hello"}
	if d.IsDuplicate("u1", model.ChannelEmail, payload) {
		t.Error("first call should not be duplicate")
	}
}

func TestDeduplicator_SecondCallIsDuplicate(t *testing.T) {
	d := New(1 * time.Minute)
	defer d.Close()

	payload := model.Payload{To: "a@b.com", Body: "hello"}
	d.IsDuplicate("u1", model.ChannelEmail, payload)

	if !d.IsDuplicate("u1", model.ChannelEmail, payload) {
		t.Error("second identical call should be duplicate")
	}
}

func TestDeduplicator_DifferentPayloadsNotDuplicate(t *testing.T) {
	d := New(1 * time.Minute)
	defer d.Close()

	p1 := model.Payload{To: "a@b.com", Body: "hello"}
	p2 := model.Payload{To: "a@b.com", Body: "world"}

	d.IsDuplicate("u1", model.ChannelEmail, p1)
	if d.IsDuplicate("u1", model.ChannelEmail, p2) {
		t.Error("different payloads should not be duplicate")
	}
}

func TestDeduplicator_DifferentChannelsNotDuplicate(t *testing.T) {
	d := New(1 * time.Minute)
	defer d.Close()

	payload := model.Payload{To: "a@b.com", Body: "hello"}
	d.IsDuplicate("u1", model.ChannelEmail, payload)

	if d.IsDuplicate("u1", model.ChannelSMS, payload) {
		t.Error("different channels should not be duplicate")
	}
}

func TestDeduplicator_ExpiresAfterTTL(t *testing.T) {
	d := New(50 * time.Millisecond)
	defer d.Close()

	payload := model.Payload{To: "x", Body: "y"}
	d.IsDuplicate("u1", model.ChannelPush, payload)

	time.Sleep(100 * time.Millisecond)

	if d.IsDuplicate("u1", model.ChannelPush, payload) {
		t.Error("should not be duplicate after TTL expires")
	}
}

func TestDeduplicator_ConcurrentAccess(t *testing.T) {
	d := New(1 * time.Minute)
	defer d.Close()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			payload := model.Payload{To: "x", Body: "y"}
			d.IsDuplicate("u1", model.ChannelEmail, payload)
		}(i)
	}
	wg.Wait()
}
