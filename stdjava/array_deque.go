package stdjava

// ArrayDeque owns Object slots independently of its compile-time element view.
// A raw write is legal; generated consumers check logical E after queue effects.
// Generic factories retain source inference without specializing physical views.
type ArrayDeque struct {
	elements   []any
	head, size int
}

func NewArrayDeque[T any]() *ArrayDeque { return &ArrayDeque{elements: make([]any, 17)} }
func NewArrayDequeWithCapacity[T any](capacity int32) *ArrayDeque {
	length := 1
	if capacity > 0 {
		length = int(capacity)
		if capacity != 2147483647 {
			length++
		}
	}
	return &ArrayDeque{elements: make([]any, length)}
}
func NewArrayDequeFromCollection[T any](source JavaIterable, executions ...*Execution) *ArrayDeque {
	e := optionalComparisonExecution(executions)
	if e == nil {
		e = NewExecution()
	}
	ReferenceRequireNonNull(source)
	d := NewArrayDequeWithCapacity[T](CollectionSizeExecution(e, source))
	d.copyElements(e, source)
	return d
}
func (d *ArrayDeque) check()             { ReferenceRequireNonNull(d) }
func (d *ArrayDeque) slot(index int) int { return (d.head + index) % len(d.elements) }
func (d *ArrayDeque) tail() int {
	if len(d.elements) == 0 {
		return 0
	}
	return (d.head + d.size) % len(d.elements)
}
func (d *ArrayDeque) initializeStorage() {
	if len(d.elements) == 0 {
		d.elements = make([]any, 17)
	}
}

// Grow retains the JDK ring's physical first/second legs. Iterators and a
// reentrant bulk-removal predicate can hold indices into the previous ring.
func (d *ArrayDeque) grow(needed int) {
	d.initializeStorage()
	oldCapacity := len(d.elements)
	oldTail := d.tail()
	jump := oldCapacity >> 1
	if oldCapacity < 64 {
		jump = oldCapacity + 2
	}
	newCapacity := oldCapacity + jump
	if jump < needed {
		newCapacity = oldCapacity + needed
	}
	const maxArraySize = int64(2147483639)
	if int64(newCapacity) > maxArraySize {
		minimum := int64(oldCapacity) + int64(needed)
		if minimum > 2147483647 {
			panic(NewIllegalStateException("Sorry, deque too big"))
		}
		if minimum > maxArraySize {
			newCapacity = 2147483647
		} else {
			newCapacity = int(maxArraySize)
		}
	}
	values := make([]any, newCapacity)
	copy(values, d.elements)
	if oldTail < d.head || oldTail == d.head && !javaReferenceIsNull(values[d.head]) {
		space := newCapacity - oldCapacity
		copy(values[d.head+space:], values[d.head:oldCapacity])
		clear(values[d.head : d.head+space])
		d.head += space
	}
	d.elements = values
}
func (d *ArrayDeque) addFirst(value any) {
	d.check()
	ReferenceRequireNonNull(value)
	d.initializeStorage()
	d.head = (d.head + len(d.elements) - 1) % len(d.elements)
	d.elements[d.head] = value
	d.size++
	if d.size == len(d.elements) {
		d.grow(1)
	}
}
func (d *ArrayDeque) addLast(value any) {
	d.check()
	ReferenceRequireNonNull(value)
	d.initializeStorage()
	d.elements[d.slot(d.size)] = value
	d.size++
	if d.size == len(d.elements) {
		d.grow(1)
	}
}
func (d *ArrayDeque) poll(first bool) any {
	d.check()
	if d.size == 0 {
		return nil
	}
	index := d.slot(d.size - 1)
	if first {
		index = d.head
	}
	value := d.elements[index]
	d.elements[index] = nil
	d.size--
	if first {
		d.head = (d.head + 1) % len(d.elements)
	}
	return value
}
func (d *ArrayDeque) peek(first bool) any {
	d.check()
	if d.size == 0 {
		return nil
	}
	if first {
		return d.elements[d.head]
	}
	return d.elements[d.slot(d.size-1)]
}
func (d *ArrayDeque) removeEnd(first bool) any {
	value := d.poll(first)
	if value == nil {
		panic(NewNoSuchElementException(""))
	}
	return value
}
func (d *ArrayDeque) getEnd(first bool) any {
	value := d.peek(first)
	if value == nil {
		panic(NewNoSuchElementException(""))
	}
	return value
}
func (d *ArrayDeque) AddFirst(value any)        { d.addFirst(value) }
func (d *ArrayDeque) AddLast(value any)         { d.addLast(value) }
func (d *ArrayDeque) OfferFirst(value any) bool { d.addFirst(value); return true }
func (d *ArrayDeque) OfferLast(value any) bool  { d.addLast(value); return true }
func (d *ArrayDeque) RemoveFirst() any          { return d.removeEnd(true) }
func (d *ArrayDeque) RemoveLast() any           { return d.removeEnd(false) }
func (d *ArrayDeque) PollFirst() any            { return d.poll(true) }
func (d *ArrayDeque) PollLast() any             { return d.poll(false) }
func (d *ArrayDeque) GetFirst() any             { return d.getEnd(true) }
func (d *ArrayDeque) GetLast() any              { return d.getEnd(false) }
func (d *ArrayDeque) PeekFirst() any            { return d.peek(true) }
func (d *ArrayDeque) PeekLast() any             { return d.peek(false) }
func (d *ArrayDeque) Add(value any) bool        { d.addLast(value); return true }
func (d *ArrayDeque) Offer(value any) bool      { return d.Add(value) }
func (d *ArrayDeque) Remove() any               { return d.RemoveFirst() }
func (d *ArrayDeque) Poll() any                 { return d.PollFirst() }
func (d *ArrayDeque) Element() any              { return d.GetFirst() }
func (d *ArrayDeque) Peek() any                 { return d.PeekFirst() }
func (d *ArrayDeque) Push(value any)            { d.AddFirst(value) }
func (d *ArrayDeque) Pop() any                  { return d.RemoveFirst() }
func (d *ArrayDeque) Size() int32               { d.check(); return int32(d.size) }
func (d *ArrayDeque) IsEmpty() bool             { return d.Size() == 0 }
func (d *ArrayDeque) Clear() {
	d.check()
	for i := range d.elements {
		d.elements[i] = nil
	}
	d.head, d.size = 0, 0
}
func (d *ArrayDeque) Clone() *ArrayDeque {
	d.check()
	copy := *d
	copy.elements = append([]any(nil), d.elements...)
	return &copy
}

// Delete closes the shorter side of the ring. The result identifies a shift
// toward the head, which lets forward/descending iterators retain their cursor.
func (d *ArrayDeque) deleteSlot(physical int) bool {
	index := (physical - d.head + len(d.elements)) % len(d.elements)
	if index >= d.size {
		panic(NewConcurrentModificationException(""))
	}
	if index < d.size-index-1 {
		for i := index; i > 0; i-- {
			d.elements[d.slot(i)] = d.elements[d.slot(i-1)]
		}
		d.elements[d.head] = nil
		d.head = (d.head + 1) % len(d.elements)
		d.size--
		return false
	}
	for i := index; i < d.size-1; i++ {
		d.elements[d.slot(i)] = d.elements[d.slot(i+1)]
	}
	d.elements[d.slot(d.size-1)] = nil
	d.size--
	return true
}
