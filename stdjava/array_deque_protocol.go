package stdjava

// These Go protocols represent already-proven Java references. ObjectView checks
// canonical nominal membership before a generated conversion requests one.
type JavaQueue interface {
	JavaIterable
	QueueAddJava2goExecution(*Execution, any) bool
	QueueOfferJava2goExecution(*Execution, any) bool
	QueueRemoveJava2goExecution(*Execution) any
	QueuePollJava2goExecution(*Execution) any
	QueueElementJava2goExecution(*Execution) any
	QueuePeekJava2goExecution(*Execution) any
}
type JavaDeque interface {
	JavaQueue
	DequeAddFirstJava2goExecution(*Execution, any)
	DequeOfferFirstJava2goExecution(*Execution, any) bool
	DequeRemoveFirstJava2goExecution(*Execution) any
	DequePollFirstJava2goExecution(*Execution) any
	DequeGetFirstJava2goExecution(*Execution) any
	DequePeekFirstJava2goExecution(*Execution) any
	DequeAddLastJava2goExecution(*Execution, any)
	DequeOfferLastJava2goExecution(*Execution, any) bool
	DequeRemoveLastJava2goExecution(*Execution) any
	DequePollLastJava2goExecution(*Execution) any
	DequeGetLastJava2goExecution(*Execution) any
	DequePeekLastJava2goExecution(*Execution) any
	DequePushJava2goExecution(*Execution, any)
	DequePopJava2goExecution(*Execution) any
	DequeRemoveFirstOccurrenceJava2goExecution(*Execution, any) bool
	DequeRemoveLastOccurrenceJava2goExecution(*Execution, any) bool
	DequeDescendingIteratorJava2goExecution(*Execution) JavaIterator
}

func (d *ArrayDeque) QueueAddJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	d.addLast(value)
	return true
}
func QueueAddExecution(e *Execution, q JavaQueue, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.QueueAddJava2goExecution(e, value)
}
func (d *ArrayDeque) QueueOfferJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	d.addLast(value)
	return true
}
func QueueOfferExecution(e *Execution, q JavaQueue, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.QueueOfferJava2goExecution(e, value)
}
func (d *ArrayDeque) QueueRemoveJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.removeEnd(true)
}
func QueueRemoveExecution(e *Execution, q JavaQueue) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.QueueRemoveJava2goExecution(e)
}
func (d *ArrayDeque) QueuePollJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.poll(true)
}
func QueuePollExecution(e *Execution, q JavaQueue) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.QueuePollJava2goExecution(e)
}
func (d *ArrayDeque) QueueElementJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.getEnd(true)
}
func QueueElementExecution(e *Execution, q JavaQueue) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.QueueElementJava2goExecution(e)
}
func (d *ArrayDeque) QueuePeekJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.peek(true)
}
func QueuePeekExecution(e *Execution, q JavaQueue) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.QueuePeekJava2goExecution(e)
}
func (d *ArrayDeque) DequeAddFirstJava2goExecution(e *Execution, value any) {
	requireExecution(e)
	d.addFirst(value)
}
func DequeAddFirstExecution(e *Execution, q JavaDeque, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	q.DequeAddFirstJava2goExecution(e, value)
}
func (d *ArrayDeque) DequeOfferFirstJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	d.addFirst(value)
	return true
}
func DequeOfferFirstExecution(e *Execution, q JavaDeque, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeOfferFirstJava2goExecution(e, value)
}
func (d *ArrayDeque) DequeRemoveFirstJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.removeEnd(true)
}
func DequeRemoveFirstExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeRemoveFirstJava2goExecution(e)
}
func (d *ArrayDeque) DequePollFirstJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.poll(true)
}
func DequePollFirstExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequePollFirstJava2goExecution(e)
}
func (d *ArrayDeque) DequeGetFirstJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.getEnd(true)
}
func DequeGetFirstExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeGetFirstJava2goExecution(e)
}
func (d *ArrayDeque) DequePeekFirstJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.peek(true)
}
func DequePeekFirstExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequePeekFirstJava2goExecution(e)
}
func (d *ArrayDeque) DequeAddLastJava2goExecution(e *Execution, value any) {
	requireExecution(e)
	d.addLast(value)
}
func DequeAddLastExecution(e *Execution, q JavaDeque, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	q.DequeAddLastJava2goExecution(e, value)
}
func (d *ArrayDeque) DequeOfferLastJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	d.addLast(value)
	return true
}
func DequeOfferLastExecution(e *Execution, q JavaDeque, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeOfferLastJava2goExecution(e, value)
}
func (d *ArrayDeque) DequeRemoveLastJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.removeEnd(false)
}
func DequeRemoveLastExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeRemoveLastJava2goExecution(e)
}
func (d *ArrayDeque) DequePollLastJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.poll(false)
}
func DequePollLastExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequePollLastJava2goExecution(e)
}
func (d *ArrayDeque) DequeGetLastJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.getEnd(false)
}
func DequeGetLastExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeGetLastJava2goExecution(e)
}
func (d *ArrayDeque) DequePeekLastJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.peek(false)
}
func DequePeekLastExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequePeekLastJava2goExecution(e)
}
func (d *ArrayDeque) DequePushJava2goExecution(e *Execution, value any) {
	requireExecution(e)
	d.addFirst(value)
}
func DequePushExecution(e *Execution, q JavaDeque, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	q.DequePushJava2goExecution(e, value)
}
func (d *ArrayDeque) DequePopJava2goExecution(e *Execution) any {
	requireExecution(e)
	return d.removeEnd(true)
}
func DequePopExecution(e *Execution, q JavaDeque) any {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequePopJava2goExecution(e)
}
func (d *ArrayDeque) DequeRemoveFirstOccurrenceJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	return d.RemoveFirstOccurrence(value, e)
}
func DequeRemoveFirstOccurrenceExecution(e *Execution, q JavaDeque, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeRemoveFirstOccurrenceJava2goExecution(e, value)
}
func (d *ArrayDeque) DequeRemoveLastOccurrenceJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	return d.RemoveLastOccurrence(value, e)
}
func DequeRemoveLastOccurrenceExecution(e *Execution, q JavaDeque, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeRemoveLastOccurrenceJava2goExecution(e, value)
}
func (d *ArrayDeque) DequeDescendingIteratorJava2goExecution(e *Execution) JavaIterator {
	requireExecution(e)
	return d.DescendingIteratorJava2goExecution(e)
}
func DequeDescendingIteratorExecution(e *Execution, q JavaDeque) JavaIterator {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	return q.DequeDescendingIteratorJava2goExecution(e)
}

func ArrayDequeCollectionAddAllExecution(e *Execution, q JavaIterable, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	if native, ok := q.(interface {
		CollectionAddAllJava2goExecution(*Execution, JavaIterable) bool
	}); ok {
		return native.CollectionAddAllJava2goExecution(e, other)
	}
	ReferenceRequireNonNull(other)
	changed := false
	for _, value := range ErasedCollectionIterationElements(e, other) {
		if CollectionAddExecution(e, q, value) {
			changed = true
		}
	}
	return changed
}
func ArrayDequeCollectionToArrayExecution(e *Execution, q JavaIterable) *ReferenceArray {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	if native, ok := q.(interface {
		CollectionToArrayJava2goExecution(*Execution) *ReferenceArray
	}); ok {
		return native.CollectionToArrayJava2goExecution(e)
	}
	values := []any{}
	for _, value := range ErasedCollectionIterationElements(e, q) {
		values = append(values, value)
	}
	return ReferenceArrayLiteral(ObjectTypeID, values...)
}
func ArrayDequeCollectionToArrayIntoExecution(e *Execution, q JavaIterable, a *ReferenceArray) *ReferenceArray {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	ReferenceRequireNonNull(a)
	if native, ok := q.(interface {
		CollectionToArrayIntoJava2goExecution(*Execution, *ReferenceArray) *ReferenceArray
	}); ok {
		return native.CollectionToArrayIntoJava2goExecution(e, a)
	}
	values := []any{}
	for _, value := range ErasedCollectionIterationElements(e, q) {
		values = append(values, value)
	}
	result := a
	if int(ReferenceArrayLength(a)) < len(values) {
		result = NewReferenceArray(len(values), a.ComponentType())
	}
	for i, v := range values {
		ReferenceArraySet(result, i, v)
	}
	if int(ReferenceArrayLength(result)) > len(values) {
		ReferenceArraySet[any](result, len(values), nil)
	}
	return result
}
func ArrayDequeCollectionStringExecution(e *Execution, q JavaIterable) *JavaString {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	if native, ok := q.(interface{ StringJava2goExecution(*Execution) *JavaString }); ok {
		return native.StringJava2goExecution(e)
	}
	panic(NewUnsupportedOperationException("collection text is unavailable"))
}

func ArrayDequeCollectionRemoveAllExecution(e *Execution, q JavaIterable, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	ReferenceRequireNonNull(other)
	if native, ok := q.(interface {
		CollectionRemoveAllJava2goExecution(*Execution, JavaIterable) bool
	}); ok {
		return native.CollectionRemoveAllJava2goExecution(e, other)
	}
	changed := false
	it := IterableIteratorExecution(e, q)
	for IteratorHasNextExecution(e, it) {
		if CollectionContainsExecution(e, other, IteratorNextExecution(e, it)) {
			IteratorRemoveExecution(e, it)
			changed = true
		}
	}
	return changed
}
func ArrayDequeCollectionRetainAllExecution(e *Execution, q JavaIterable, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(q)
	ReferenceRequireNonNull(other)
	if native, ok := q.(interface {
		CollectionRetainAllJava2goExecution(*Execution, JavaIterable) bool
	}); ok {
		return native.CollectionRetainAllJava2goExecution(e, other)
	}
	changed := false
	it := IterableIteratorExecution(e, q)
	for IteratorHasNextExecution(e, it) {
		if !CollectionContainsExecution(e, other, IteratorNextExecution(e, it)) {
			IteratorRemoveExecution(e, it)
			changed = true
		}
	}
	return changed
}
