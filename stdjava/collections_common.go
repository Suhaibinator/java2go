package stdjava

import (
	"cmp"
	"reflect"
	"sort"
	"strings"
)

// This file holds shared helpers for the collection types and the static
// utility methods of java.util.Collections and java.util.Arrays.

// ObjectsEqual reports whether two values are equal using Java-style value
// equality. It approximates Object.equals with reflect.DeepEqual, which matches
// equals() for strings, boxed numbers, and value structs; it does not invoke a
// user-defined equals() method.
func ObjectsEqual[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

// SortOrdered sorts a list in place by natural ordering, matching
// Collections.sort(List) on a Comparable element type.
//
// The element type is not constrained to cmp.Ordered because Java's natural
// ordering covers any Comparable, including a user class whose compareTo the
// transpiler generates. Element types with a direct Go ordering keep a fast
// path; everything else goes through the CompareTo bridge.
func SortOrdered[T any](l *List[T]) {
	if l == nil {
		panic(NewNullPointerException("Collections.sort on null"))
	}
	switch elements := any(l.elements).(type) {
	case []string:
		SortSlice(elements)
	case []int32:
		SortSlice(elements)
	case []int64:
		SortSlice(elements)
	case []int16:
		SortSlice(elements)
	case []int8:
		SortSlice(elements)
	case []float32:
		SortSlice(elements)
	case []float64:
		SortSlice(elements)
	default:
		SortSliceStableNatural(l.elements)
	}
}

// ReverseList reverses a list in place, matching Collections.reverse.
func ReverseList[T any](l *List[T]) {
	for i, j := 0, len(l.elements)-1; i < j; i, j = i+1, j-1 {
		l.elements[i], l.elements[j] = l.elements[j], l.elements[i]
	}
}

// MaxOrdered returns the largest element of a list by natural ordering, matching
// Collections.max(Collection). Like Collections.max it keeps the earlier element
// on a tie, and like SortOrdered it accepts any Comparable element type.
func MaxOrdered[T any](l *List[T]) T {
	if l == nil || len(l.elements) == 0 {
		panic(NewNoSuchElementException("Collections.max on an empty collection"))
	}
	best := l.elements[0]
	for _, e := range l.elements[1:] {
		if javaCompareValues(e, best) > 0 {
			best = e
		}
	}
	return best
}

// MinOrdered returns the smallest element of a list by natural ordering,
// matching Collections.min(Collection).
func MinOrdered[T any](l *List[T]) T {
	if l == nil || len(l.elements) == 0 {
		panic(NewNoSuchElementException("Collections.min on an empty collection"))
	}
	best := l.elements[0]
	for _, e := range l.elements[1:] {
		if javaCompareValues(e, best) < 0 {
			best = e
		}
	}
	return best
}

// EmptyList returns a new empty List, matching Collections.emptyList. The result
// is mutable (the Java immutability guarantee is not modelled).
func EmptyList[T any]() *List[T] {
	return NewList[T]()
}

// SingletonList returns a List containing one element, matching
// Collections.singletonList.
func SingletonList[T any](element T) *List[T] {
	return NewListFrom(element)
}

// UnmodifiableList returns the list unchanged, approximating
// Collections.unmodifiableList. The read-only guarantee is not enforced.
func UnmodifiableList[T any](l *List[T]) *List[T] {
	return l
}

// AsList returns a List backed by the given elements, matching Arrays.asList.
func AsList[T any](elements ...T) *List[T] {
	return NewListFrom(elements...)
}

// SortSlice sorts a slice of ordered elements in place, matching Arrays.sort.
func SortSlice[T cmp.Ordered](elements []T) {
	sort.Slice(elements, func(i, j int) bool {
		return elements[i] < elements[j]
	})
}

// SortArray is the generated Arrays.sort bridge for descriptor-bearing Java
// arrays. Primitive wrappers retain a direct native-slice sort; reference
// arrays support the value-backed comparable types currently emitted by the
// transpiler (String and boxed primitive scalars).
func SortArray(array any) {
	switch values := array.(type) {
	case *PrimitiveArray[int8]:
		SortSlice(values.Elements)
	case *PrimitiveArray[int16]:
		SortSlice(values.Elements)
	case *PrimitiveArray[int32]:
		SortSlice(values.Elements)
	case *PrimitiveArray[int64]:
		SortSlice(values.Elements)
	case *PrimitiveArray[float32]:
		SortSlice(values.Elements)
	case *PrimitiveArray[float64]:
		SortSlice(values.Elements)
	case *ReferenceArray:
		if values == nil {
			panic(NewNullPointerException("Arrays.sort on null"))
		}
		sort.SliceStable(values.elements, func(i, j int) bool {
			return javaComparableLess(values.elements[i], values.elements[j])
		})
	default:
		reflected := reflect.ValueOf(array)
		if !reflected.IsValid() || ((reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Slice) && reflected.IsNil()) {
			panic(NewNullPointerException("Arrays.sort on null"))
		}
		if reflected.Kind() != reflect.Slice {
			panic(NewIllegalArgumentException("Arrays.sort requires an array"))
		}
		sort.Slice(array, func(i, j int) bool {
			return javaComparableLess(reflected.Index(i).Interface(), reflected.Index(j).Interface())
		})
	}
}

// javaComparableLess reports whether left sorts before right under Java's
// natural ordering. Values that are not one of the directly-handled primitive or
// String forms are bridged to their own generated CompareTo method, so a user
// class implementing Comparable sorts correctly through Arrays.sort and the
// Collections utilities.
func javaComparableLess(left, right any) bool {
	return javaCompareValues(left, right) < 0
}

// SliceToString returns the Java Arrays.toString form of a slice, e.g.
// "[a, b, c]".
func SliceToString[T any](elements []T) string {
	parts := make([]string, len(elements))
	for i, e := range elements {
		parts[i] = StringValueOf(e)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ArrayToString is the descriptor-bearing Arrays.toString bridge. Unlike
// deepToString it renders one level and applies String.valueOf to each element.
func ArrayToString(value any) string {
	if value == nil {
		return "null"
	}
	var elements []any
	switch array := value.(type) {
	case *ReferenceArray:
		if array == nil {
			return "null"
		}
		elements = array.elements
	case *PrimitiveArray[bool]:
		return primitiveArrayToString(array)
	case *PrimitiveArray[int8]:
		return primitiveArrayToString(array)
	case *PrimitiveArray[int16]:
		return primitiveArrayToString(array)
	case *PrimitiveArray[int32]:
		return primitiveArrayToString(array)
	case *PrimitiveArray[int64]:
		return primitiveArrayToString(array)
	case *PrimitiveArray[float32]:
		return primitiveArrayToString(array)
	case *PrimitiveArray[float64]:
		return primitiveArrayToString(array)
	default:
		reflected := reflect.ValueOf(value)
		if (reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Slice) && reflected.IsNil() {
			return "null"
		}
		if reflected.Kind() != reflect.Array && reflected.Kind() != reflect.Slice {
			return StringValueOf(value)
		}
		elements = make([]any, reflected.Len())
		for index := range elements {
			elements[index] = reflected.Index(index).Interface()
		}
	}
	parts := make([]string, len(elements))
	for index, element := range elements {
		parts[index] = StringValueOf(element)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func primitiveArrayToString[T any](array *PrimitiveArray[T]) string {
	if array == nil {
		return "null"
	}
	return SliceToString(array.Elements)
}

// ArrayDeepToString returns the recursive Java Arrays.deepToString form of a
// nested array. Slices model Java arrays in generated Go, including primitive
// arrays nested inside an Object-style outer array. Active recursion is tracked
// so a self-referential array is rendered as "[...]" rather than recursing
// forever, matching java.util.Arrays.
func ArrayDeepToString(value any) string {
	var out strings.Builder
	appendDeepArrayString(&out, reflect.ValueOf(value), make(map[deepArrayVisit]bool))
	return out.String()
}

type deepArrayVisit struct {
	typeOf  reflect.Type
	pointer uintptr
}

func appendDeepArrayString(out *strings.Builder, value reflect.Value, active map[deepArrayVisit]bool) {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			out.WriteString("null")
			return
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		out.WriteString("null")
		return
	}
	if value.CanInterface() {
		switch array := value.Interface().(type) {
		case *ReferenceArray:
			appendReferenceArrayString(out, array, active)
			return
		case *PrimitiveArray[bool]:
			appendPrimitiveArrayString(out, array, active)
			return
		case *PrimitiveArray[int8]:
			appendPrimitiveArrayString(out, array, active)
			return
		case *PrimitiveArray[int16]:
			appendPrimitiveArrayString(out, array, active)
			return
		case *PrimitiveArray[int32]:
			appendPrimitiveArrayString(out, array, active)
			return
		case *PrimitiveArray[int64]:
			appendPrimitiveArrayString(out, array, active)
			return
		case *PrimitiveArray[float32]:
			appendPrimitiveArrayString(out, array, active)
			return
		case *PrimitiveArray[float64]:
			appendPrimitiveArrayString(out, array, active)
			return
		}
	}

	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		if (value.Kind() == reflect.Pointer || value.Kind() == reflect.Map || value.Kind() == reflect.Func) && value.IsNil() {
			out.WriteString("null")
			return
		}
		out.WriteString(StringValueOf(value.Interface()))
		return
	}

	if value.Kind() == reflect.Slice {
		if value.IsNil() {
			out.WriteString("null")
			return
		}
		visit := deepArrayVisit{typeOf: value.Type(), pointer: value.Pointer()}
		if active[visit] {
			out.WriteString("[...]")
			return
		}
		active[visit] = true
		defer delete(active, visit)
	}

	out.WriteByte('[')
	for index := 0; index < value.Len(); index++ {
		if index > 0 {
			out.WriteString(", ")
		}
		appendDeepArrayString(out, value.Index(index), active)
	}
	out.WriteByte(']')
}

func appendReferenceArrayString(out *strings.Builder, array *ReferenceArray, active map[deepArrayVisit]bool) {
	if array == nil {
		out.WriteString("null")
		return
	}
	visit := deepArrayVisit{typeOf: reflect.TypeOf(array), pointer: reflect.ValueOf(array).Pointer()}
	if active[visit] {
		out.WriteString("[...]")
		return
	}
	active[visit] = true
	defer delete(active, visit)
	out.WriteByte('[')
	for index, element := range array.elements {
		if index > 0 {
			out.WriteString(", ")
		}
		appendDeepArrayString(out, reflect.ValueOf(element), active)
	}
	out.WriteByte(']')
}

func appendPrimitiveArrayString[T any](out *strings.Builder, array *PrimitiveArray[T], active map[deepArrayVisit]bool) {
	if array == nil {
		out.WriteString("null")
		return
	}
	visit := deepArrayVisit{typeOf: reflect.TypeOf(array), pointer: reflect.ValueOf(array).Pointer()}
	if active[visit] {
		out.WriteString("[...]")
		return
	}
	active[visit] = true
	defer delete(active, visit)
	out.WriteByte('[')
	for index, element := range array.Elements {
		if index > 0 {
			out.WriteString(", ")
		}
		appendDeepArrayString(out, reflect.ValueOf(element), active)
	}
	out.WriteByte(']')
}
