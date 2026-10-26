package stdjava

import "fmt"

func (d *ArrayDeque) copyElements(e *Execution, source JavaIterable) {
	ReferenceRequireNonNull(source)
	// Preserve a source-defined forEach override when its canonical bridge exists.
	if custom, ok := source.(interface{ CollectionForEachJava2goExecution(*Execution, any) }); ok {
		custom.CollectionForEachJava2goExecution(e, NewConsumerFuncAdapter[any](func(_ *Execution, value any) { d.addLast(value) }))
		return
	}
	for _, value := range ErasedCollectionIterationElements(e, source) {
		d.addLast(value)
	}
}
func (d *ArrayDeque) AddAll(source JavaIterable, executions ...*Execution) bool {
	d.check()
	ReferenceRequireNonNull(source)
	e := optionalComparisonExecution(executions)
	if e == nil {
		e = NewExecution()
	}
	// JDK observes size before invoking source forEach; failures preserve prior writes.
	needed := int32(d.size) + CollectionSizeExecution(e, source) + 1 - int32(len(d.elements))
	if needed > 0 {
		d.grow(int(needed))
	}
	before := d.size
	d.copyElements(e, source)
	return d.size > before
}
func (d *ArrayDeque) Contains(value any, executions ...*Execution) bool {
	d.check()
	if javaReferenceIsNull(value) {
		return false
	}
	for i := 0; i < d.size; i++ {
		if ObjectsEqual(value, d.elements[d.slot(i)], executions...) {
			return true
		}
	}
	return false
}
func (d *ArrayDeque) removeOccurrence(value any, last bool, executions ...*Execution) bool {
	d.check()
	if javaReferenceIsNull(value) {
		return false
	}
	for index := 0; index < d.size; index++ {
		i := index
		if last {
			i = d.size - index - 1
		}
		physical := d.slot(i)
		if ObjectsEqual(value, d.elements[physical], executions...) {
			d.deleteSlot(physical)
			return true
		}
	}
	return false
}
func (d *ArrayDeque) RemoveFirstOccurrence(value any, e ...*Execution) bool {
	return d.removeOccurrence(value, false, e...)
}
func (d *ArrayDeque) RemoveLastOccurrence(value any, e ...*Execution) bool {
	return d.removeOccurrence(value, true, e...)
}
func (d *ArrayDeque) RemoveObject(value any, e ...*Execution) bool {
	return d.RemoveFirstOccurrence(value, e...)
}
func (d *ArrayDeque) ContainsAll(other JavaIterable, executions ...*Execution) bool {
	d.check()
	ReferenceRequireNonNull(other)
	e := optionalComparisonExecution(executions)
	if e == nil {
		e = NewExecution()
	}
	for _, value := range ErasedCollectionIterationElements(e, other) {
		if !d.Contains(value, e) {
			return false
		}
	}
	return true
}
func (d *ArrayDeque) RemoveAll(other JavaIterable, executions ...*Execution) bool {
	return d.bulkRemove(other, false, executions...)
}
func (d *ArrayDeque) RetainAll(other JavaIterable, executions ...*Execution) bool {
	return d.bulkRemove(other, true, executions...)
}
func (d *ArrayDeque) bulkRemove(other JavaIterable, retain bool, executions ...*Execution) bool {
	d.check()
	ReferenceRequireNonNull(other)
	e := optionalComparisonExecution(executions)
	if e == nil {
		e = NewExecution()
	}
	filter := func(value any) bool { return CollectionContainsExecution(e, other, value) != retain }
	// The first scan captures its ring/end; after the first match JDK captures
	// a new ring/end for marking and compaction. A callback's writes stay visible.
	elements := d.elements
	end := d.tail()
	i := d.head
	to := end
	if i > end {
		to = len(elements)
	}
	for {
		for ; i < to; i++ {
			if filter(elements[i]) {
				return d.bulkRemoveModified(filter, i)
			}
		}
		if to == end {
			if end != d.tail() {
				panic(NewConcurrentModificationException(""))
			}
			break
		}
		i = 0
		to = end
	}
	return false
}
func arrayDequeDistance(end, start, capacity int) int {
	n := end - start
	if n < 0 {
		n += capacity
	}
	return n
}
func arrayDequeSetDeathBit(bits []uint64, index int) {
	word := index >> 6
	if word < 0 || word >= len(bits) {
		panic(NewArrayIndexOutOfBoundsException(fmt.Sprintf("Index %d out of bounds for length %d", word, len(bits))))
	}
	bits[word] |= uint64(1) << uint(index&63)
}
func (d *ArrayDeque) bulkRemoveModified(filter func(any) bool, beg int) bool {
	elements := d.elements
	capacity := len(elements)
	end := d.tail()
	count := arrayDequeDistance(end, beg, capacity)
	// Preserve Java array bounds semantics for an empty captured death row.
	// Reentrant growth can legitimately make the old beginning equal the new end.
	deathRow := make([]uint64, ((count-1)>>6)+1)
	arrayDequeSetDeathBit(deathRow, 0)
	for offset := 1; offset < count; offset++ {
		if filter(elements[(beg+offset)%capacity]) {
			arrayDequeSetDeathBit(deathRow, offset)
		}
	}
	write := beg
	for offset := 1; offset < count; offset++ {
		if deathRow[offset>>6]&(uint64(1)<<uint(offset&63)) == 0 {
			elements[write] = elements[(beg+offset)%capacity]
			write = (write + 1) % capacity
		}
	}
	if end != d.tail() {
		panic(NewConcurrentModificationException(""))
	}
	for i := write; i != end; i = (i + 1) % capacity {
		elements[i] = nil
	}
	d.size = arrayDequeDistance(write, d.head, len(d.elements))
	return true
}
func (d *ArrayDeque) ToArray() *ReferenceArray {
	d.check()
	result := NewReferenceArray(d.size, ObjectTypeID)
	for i := 0; i < d.size; i++ {
		ReferenceArraySet(result, i, d.elements[d.slot(i)])
	}
	return result
}
func (d *ArrayDeque) ToArrayInto(array *ReferenceArray) *ReferenceArray {
	d.check()
	ReferenceRequireNonNull(array)
	result := array
	if ReferenceArrayLength(array) < int32(d.size) {
		result = NewReferenceArray(d.size, array.ComponentType())
	}
	for i := 0; i < d.size; i++ {
		ReferenceArraySet(result, i, d.elements[d.slot(i)])
	}
	if ReferenceArrayLength(result) > int32(d.size) {
		ReferenceArraySet[any](result, d.size, nil)
	}
	return result
}

// String is a legacy host-text facade. Generated Java uses the UTF16 method.
func (d *ArrayDeque) String() string {
	return string(unsignedBytes(JavaStringGetBytes(d.StringJava2goExecution(NewExecution()), UTF_8).Elements))
}
func (d *ArrayDeque) StringJava2goExecution(e *Execution) *JavaString {
	requireExecution(e)
	d.check()
	units := []uint16{'['}
	it := d.IteratorJava2goExecution(e)
	for IteratorHasNextExecution(e, it) {
		if len(units) > 1 {
			units = append(units, ',', ' ')
		}
		value := IteratorNextExecution(e, it)
		if JavaReferenceEqual(value, d) {
			units = append(units, '(', 't', 'h', 'i', 's', ' ', 'C', 'o', 'l', 'l', 'e', 'c', 't', 'i', 'o', 'n', ')')
		} else {
			units = append(units, JavaStringTextOperandExecution(e, value).units...)
		}
	}
	units = append(units, ']')
	return &JavaString{units: units}
}
func (d *ArrayDeque) CollectionAddJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	d.addLast(value)
	return true
}
func (d *ArrayDeque) CollectionRemoveJava2goExecution(e *Execution, value any) bool {
	return d.RemoveObject(value, e)
}
func (d *ArrayDeque) CollectionContainsJava2goExecution(e *Execution, value any) bool {
	return d.Contains(value, e)
}
func (d *ArrayDeque) CollectionSizeJava2goExecution(e *Execution) int32 {
	requireExecution(e)
	return d.Size()
}
func (d *ArrayDeque) CollectionIsEmptyJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return d.IsEmpty()
}
func (d *ArrayDeque) CollectionClearJava2goExecution(e *Execution) { requireExecution(e); d.Clear() }
func (d *ArrayDeque) CollectionAddAllJava2goExecution(e *Execution, other JavaIterable) bool {
	return d.AddAll(other, e)
}
func (d *ArrayDeque) CollectionToArrayJava2goExecution(e *Execution) *ReferenceArray {
	requireExecution(e)
	return d.ToArray()
}
func (d *ArrayDeque) CollectionToArrayIntoJava2goExecution(e *Execution, array *ReferenceArray) *ReferenceArray {
	requireExecution(e)
	return d.ToArrayInto(array)
}

func (d *ArrayDeque) CollectionContainsAllJava2goExecution(e *Execution, other JavaIterable) bool {
	return d.ContainsAll(other, e)
}
func (d *ArrayDeque) CollectionRemoveAllJava2goExecution(e *Execution, other JavaIterable) bool {
	return d.RemoveAll(other, e)
}
func (d *ArrayDeque) CollectionRetainAllJava2goExecution(e *Execution, other JavaIterable) bool {
	return d.RetainAll(other, e)
}
