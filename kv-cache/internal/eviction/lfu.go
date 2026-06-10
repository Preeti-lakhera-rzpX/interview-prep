package eviction

// lfuEntry tracks a key's frequency and position in its frequency bucket.
type lfuEntry struct {
	key  string
	freq int
	prev *lfuEntry
	next *lfuEntry
}

// freqBucket is a doubly-linked list of entries with the same access frequency.
type freqBucket struct {
	head *lfuEntry // sentinel
	tail *lfuEntry // sentinel
	size int
}

func newFreqBucket() *freqBucket {
	head := &lfuEntry{}
	tail := &lfuEntry{}
	head.next = tail
	tail.prev = head
	return &freqBucket{head: head, tail: tail}
}

func (b *freqBucket) pushFront(e *lfuEntry) {
	e.next = b.head.next
	e.prev = b.head
	b.head.next.prev = e
	b.head.next = e
	b.size++
}

func (b *freqBucket) removeLast() *lfuEntry {
	e := b.tail.prev
	if e == b.head {
		return nil
	}
	b.detach(e)
	return e
}

func (b *freqBucket) detach(e *lfuEntry) {
	e.prev.next = e.next
	e.next.prev = e.prev
	b.size--
}

func (b *freqBucket) empty() bool {
	return b.size == 0
}

// lfu implements the LFU eviction policy.
// Evicts the least frequently used key; ties broken by oldest insertion.
type lfu struct {
	capacity int
	items    map[string]*lfuEntry
	buckets  map[int]*freqBucket
	minFreq  int
}

func newLFU(capacity int) *lfu {
	return &lfu{
		capacity: capacity,
		items:    make(map[string]*lfuEntry, capacity),
		buckets:  make(map[int]*freqBucket),
	}
}

func (l *lfu) Access(key string) {
	entry, ok := l.items[key]
	if !ok {
		return
	}
	l.incrementFreq(entry)
}

func (l *lfu) Add(key string) (evicted string) {
	if _, ok := l.items[key]; ok {
		return ""
	}
	if len(l.items) >= l.capacity {
		evicted = l.evictMin()
	}
	entry := &lfuEntry{key: key, freq: 1}
	l.items[key] = entry
	l.getBucket(1).pushFront(entry)
	l.minFreq = 1
	return evicted
}

func (l *lfu) Remove(key string) {
	entry, ok := l.items[key]
	if !ok {
		return
	}
	bucket := l.buckets[entry.freq]
	bucket.detach(entry)
	if bucket.empty() && entry.freq == l.minFreq {
		l.minFreq++
	}
	delete(l.items, key)
}

func (l *lfu) Len() int {
	return len(l.items)
}

func (l *lfu) incrementFreq(entry *lfuEntry) {
	oldBucket := l.buckets[entry.freq]
	oldBucket.detach(entry)
	if oldBucket.empty() && entry.freq == l.minFreq {
		l.minFreq++
	}
	entry.freq++
	l.getBucket(entry.freq).pushFront(entry)
}

func (l *lfu) evictMin() string {
	bucket := l.buckets[l.minFreq]
	if bucket == nil {
		return ""
	}
	entry := bucket.removeLast()
	if entry == nil {
		return ""
	}
	delete(l.items, entry.key)
	return entry.key
}

func (l *lfu) getBucket(freq int) *freqBucket {
	if b, ok := l.buckets[freq]; ok {
		return b
	}
	b := newFreqBucket()
	l.buckets[freq] = b
	return b
}
