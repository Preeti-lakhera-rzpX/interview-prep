package eviction

// fifo implements the FIFO eviction policy.
// Evicts the oldest inserted key regardless of access pattern.
type fifo struct {
	capacity int
	items    map[string]*fifoNode
	head     *fifoNode // sentinel, oldest at head.next
	tail     *fifoNode // sentinel, newest at tail.prev
}

type fifoNode struct {
	key  string
	prev *fifoNode
	next *fifoNode
}

func newFIFO(capacity int) *fifo {
	head := &fifoNode{}
	tail := &fifoNode{}
	head.next = tail
	tail.prev = head
	return &fifo{
		capacity: capacity,
		items:    make(map[string]*fifoNode, capacity),
		head:     head,
		tail:     tail,
	}
}

func (f *fifo) Access(key string) {
	// FIFO ignores access — eviction order is purely insertion order.
}

func (f *fifo) Add(key string) (evicted string) {
	if _, ok := f.items[key]; ok {
		return ""
	}
	if len(f.items) >= f.capacity {
		evicted = f.evictOldest()
	}
	node := &fifoNode{key: key}
	f.items[key] = node
	f.pushBack(node)
	return evicted
}

func (f *fifo) Remove(key string) {
	node, ok := f.items[key]
	if !ok {
		return
	}
	f.detach(node)
	delete(f.items, key)
}

func (f *fifo) Len() int {
	return len(f.items)
}

func (f *fifo) pushBack(node *fifoNode) {
	node.prev = f.tail.prev
	node.next = f.tail
	f.tail.prev.next = node
	f.tail.prev = node
}

func (f *fifo) detach(node *fifoNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (f *fifo) evictOldest() string {
	node := f.head.next
	if node == f.tail {
		return ""
	}
	f.detach(node)
	delete(f.items, node.key)
	return node.key
}
