package eviction

// lruNode is a doubly-linked list node.
type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

// lru implements the LRU eviction policy using a doubly-linked list and map.
// Most recently accessed items are at the front; eviction happens from the back.
type lru struct {
	capacity int
	items    map[string]*lruNode
	head     *lruNode // sentinel
	tail     *lruNode // sentinel
}

func newLRU(capacity int) *lru {
	head := &lruNode{}
	tail := &lruNode{}
	head.next = tail
	tail.prev = head
	return &lru{
		capacity: capacity,
		items:    make(map[string]*lruNode, capacity),
		head:     head,
		tail:     tail,
	}
}

func (l *lru) Access(key string) {
	node, ok := l.items[key]
	if !ok {
		return
	}
	l.detach(node)
	l.pushFront(node)
}

func (l *lru) Add(key string) (evicted string) {
	if _, ok := l.items[key]; ok {
		return ""
	}
	if len(l.items) >= l.capacity {
		evicted = l.evictBack()
	}
	node := &lruNode{key: key}
	l.items[key] = node
	l.pushFront(node)
	return evicted
}

func (l *lru) Remove(key string) {
	node, ok := l.items[key]
	if !ok {
		return
	}
	l.detach(node)
	delete(l.items, key)
}

func (l *lru) Len() int {
	return len(l.items)
}

func (l *lru) detach(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (l *lru) pushFront(node *lruNode) {
	node.next = l.head.next
	node.prev = l.head
	l.head.next.prev = node
	l.head.next = node
}

func (l *lru) evictBack() string {
	node := l.tail.prev
	if node == l.head {
		return ""
	}
	l.detach(node)
	delete(l.items, node.key)
	return node.key
}
