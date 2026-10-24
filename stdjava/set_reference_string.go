package stdjava

// StringJava2goExecution implements AbstractCollection.toString for sets at
// the canonical String boundary. Use the set's existing iterator so sorted
// order, encounter order and structural callback effects stay observable.
func (set *Set[T]) StringJava2goExecution(execution *Execution) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	cursor := set.IteratorJava2goExecution(execution)
	if !IteratorHasNextExecution(execution, cursor) {
		return JavaStringLiteralUTF16([]uint16{'[', ']'})
	}
	units := []uint16{'['}
	for {
		element := IteratorNextExecution(execution, cursor)
		if JavaReferenceEqual(element, set) {
			units = append(units, '(', 't', 'h', 'i', 's', ' ', 'C', 'o', 'l', 'l', 'e', 'c', 't', 'i', 'o', 'n', ')')
		} else {
			units = append(units, JavaStringTextOperandExecution(execution, element).units...)
		}
		if !IteratorHasNextExecution(execution, cursor) {
			break
		}
		units = append(units, ',', ' ')
	}
	units = append(units, ']')
	return &JavaString{units: units}
}
