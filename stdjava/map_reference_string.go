package stdjava

// StringJava2goExecution implements AbstractMap.toString at the canonical
// String boundary. Keep the live entry cursor and copy text as UTF16 units;
// source callbacks receive the caller execution and retain virtual dispatch.
func (m *Map[K, V]) StringJava2goExecution(execution *Execution) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	cursor := MapEntriesView(m).IteratorJava2goExecution(execution)
	if !IteratorHasNextExecution(execution, cursor) {
		return JavaStringLiteralUTF16([]uint16{'{', '}'})
	}
	units := []uint16{'{'}
	for {
		entry := IteratorNextExecution(execution, cursor).(JavaMapEntry)
		// AbstractMap reads both references before converting either to text. A
		// key's toString may replace this entry's value, but this iteration still
		// renders the previously fetched reference; later entries remain live.
		key := MapEntryGetKeyExecution(execution, entry)
		value := MapEntryGetValueExecution(execution, entry)
		for index, element := range []any{key, value} {
			if index != 0 {
				units = append(units, '=')
			}
			if JavaReferenceEqual(element, m) {
				units = append(units, '(', 't', 'h', 'i', 's', ' ', 'M', 'a', 'p', ')')
			} else {
				units = append(units, JavaStringTextOperandExecution(execution, element).units...)
			}
		}
		if !IteratorHasNextExecution(execution, cursor) {
			break
		}
		units = append(units, ',', ' ')
	}
	units = append(units, '}')
	return &JavaString{units: units}
}
