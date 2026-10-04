package stdjava

// StringJava2goExecution implements AbstractCollection.toString at the
// canonical String boundary. Read through the erased fail-fast iterator so
// callbacks observe live slots and structural changes retain Java's timing.
// Element text is converted once with the caller's execution and copied as
// UTF16 units, without a host encoding or formatter boundary.
func (list *List[T]) StringJava2goExecution(execution *Execution) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(list)
	cursor := list.IteratorJava2goExecution(execution)
	if !IteratorHasNextExecution(execution, cursor) {
		// AbstractCollection returns the literal for an empty iterator.
		return JavaStringLiteralUTF16([]uint16{'[', ']'})
	}
	units := []uint16{'['}
	for {
		element := IteratorNextExecution(execution, cursor)
		if JavaReferenceEqual(element, list) {
			units = append(units, '(', 't', 'h', 'i', 's', ' ', 'C', 'o', 'l', 'l', 'e', 'c', 't', 'i', 'o', 'n', ')')
		} else {
			text := JavaStringTextOperandExecution(execution, element)
			units = append(units, text.units...)
		}
		if !IteratorHasNextExecution(execution, cursor) {
			break
		}
		units = append(units, ',', ' ')
	}
	units = append(units, ']')
	return &JavaString{units: units}
}
