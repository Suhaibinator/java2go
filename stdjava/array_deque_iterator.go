package stdjava

type arrayDequeIterator struct {
	source                  *ArrayDeque
	cursor, remaining, last int
	descending              bool
}

func (d *ArrayDeque) IteratorJava2goExecution(e *Execution) JavaIterator {
	requireExecution(e)
	d.check()
	return &arrayDequeIterator{source: d, cursor: d.head, remaining: d.size, last: -1}
}
func (d *ArrayDeque) DescendingIteratorJava2goExecution(e *Execution) JavaIterator {
	requireExecution(e)
	d.check()
	cursor := 0
	if len(d.elements) > 0 {
		cursor = (d.head + d.size + len(d.elements) - 1) % len(d.elements)
	}
	return &arrayDequeIterator{source: d, cursor: cursor, remaining: d.size, last: -1, descending: true}
}
func (it *arrayDequeIterator) HasNextJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return it.remaining > 0
}
func (it *arrayDequeIterator) NextJava2goExecution(e *Execution) any {
	requireExecution(e)
	if it.remaining <= 0 {
		panic(NewNoSuchElementException(""))
	}
	values := it.source.elements
	if it.cursor >= len(values) || javaReferenceIsNull(values[it.cursor]) {
		panic(NewConcurrentModificationException(""))
	}
	value := values[it.cursor]
	it.last = it.cursor
	step := 1
	if it.descending {
		step = -1
	}
	it.cursor = (it.cursor + step + len(values)) % len(values)
	it.remaining--
	return value
}
func (it *arrayDequeIterator) IteratorRemoveJava2goExecution(e *Execution) {
	requireExecution(e)
	if it.last < 0 {
		panic(NewIllegalStateException(""))
	}
	shifted := it.source.deleteSlot(it.last)
	if shifted && !it.descending {
		it.cursor = (it.cursor + len(it.source.elements) - 1) % len(it.source.elements)
	}
	if !shifted && it.descending {
		it.cursor = (it.cursor + 1) % len(it.source.elements)
	}
	it.last = -1
}
