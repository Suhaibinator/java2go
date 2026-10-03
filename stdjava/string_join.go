package stdjava

import "strings"

// StringJoinIterableExecution follows the JDK Iterable overload: both arguments
// are checked before delimiter conversion, and iterator creation happens after
// that conversion. Each erased next result is checked at this consumer boundary.
func StringJoinIterableExecution(execution *Execution, delimiter, elements any) string {
	requireExecution(execution)
	ReferenceRequireNonNull(delimiter)
	ReferenceRequireNonNull(elements)
	separator := StringValueOfExecution(execution, delimiter)
	iterable, ok := elements.(JavaIterable)
	if !ok {
		panic(NewUnsupportedOperationException("Iterable iterator protocol is not implemented"))
	}
	iterator := IterableIteratorExecution(execution, iterable)
	var parts []string
	for IteratorHasNextExecution(execution, iterator) {
		element := IteratorNextExecution(execution, iterator)
		element = stringJoinCharSequence(element)
		parts = append(parts, StringValueOfExecution(execution, element))
	}
	return stringJoinConverted(separator, parts)
}

// The array overload converts its delimiter before dereferencing the array.
// Slots are fetched just before conversion so preceding callbacks can replace
// later elements. The source array, including its null identity, is retained.
func StringJoinArrayExecution(execution *Execution, delimiter any, elements *ReferenceArray) string {
	requireExecution(execution)
	ReferenceRequireNonNull(delimiter)
	separator := StringValueOfExecution(execution, delimiter)
	length := ReferenceArrayLength(elements)
	parts := make([]string, 0, length)
	for index := int32(0); index < length; index++ {
		element := ReferenceArrayGet[any](elements, index, CharSequenceTypeID)
		parts = append(parts, StringValueOfExecution(execution, element))
	}
	return stringJoinConverted(separator, parts)
}

// Expanded varargs are evaluated once by the call before entering this helper.
func StringJoinValuesExecution(execution *Execution, delimiter any, elements ...any) string {
	requireExecution(execution)
	ReferenceRequireNonNull(delimiter)
	separator := StringValueOfExecution(execution, delimiter)
	parts := make([]string, 0, len(elements))
	for _, element := range elements {
		parts = append(parts, StringValueOfExecution(execution, stringJoinCharSequence(element)))
	}
	return stringJoinConverted(separator, parts)
}

func stringJoinCharSequence(element any) any {
	if javaReferenceIsNull(element) {
		return nil
	}
	// This runtime class has an existing shared StringBuilder/StringBuffer ABI;
	// both Java declarations implement CharSequence. Source classes must instead
	// prove nominal assignability, never merely expose similarly named methods.
	if _, ok := element.(*StringBuilder); ok {
		return element
	}
	return ObjectView[any](element, CharSequenceTypeID)
}

// JDK join completes all conversions before checking the converted strings.
// A null-returning toString differs from a null element (which becomes "null").
// A null separator string is dereferenced only when there are >=2 elements.
func stringJoinConverted(separator string, parts []string) string {
	if len(parts) > 1 {
		StringRequireNonNull(separator)
	}
	for _, part := range parts {
		StringRequireNonNull(part)
	}
	return strings.Join(parts, separator)
}
