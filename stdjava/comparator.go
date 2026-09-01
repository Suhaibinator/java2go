package stdjava

import (
	"cmp"
	"reflect"
	"sort"
)

// This file implements java.util.Comparator, the functional interface behind
// every comparator-taking sort, min, and max in the Java standard library.
//
// A Java `Comparator<T>` lambda is lowered to a plain `func(a, b T) int32`, and
// Go's assignability rules for named types mean such a literal is usable
// wherever a Comparator[T] is expected without an explicit conversion.
//
// Documented approximations:
//   - Comparator.equals(Object) is not modelled; comparators are Go funcs and
//     carry no identity beyond the closure itself.
//   - nullsFirst/nullsLast are not modelled, since the transpiler has no single
//     null representation shared by every element type.

// Comparator is java.util.Comparator<T>: a two-argument ordering function whose
// result is negative, zero, or positive as the first argument sorts before,
// equal to, or after the second.
type Comparator[T any] func(a, b T) int32

// NaturalOrder returns the comparator for a type's own ordering, matching
// Comparator.naturalOrder().
func NaturalOrder[T cmp.Ordered]() Comparator[T] {
	return func(a, b T) int32 {
		return int32(cmp.Compare(a, b))
	}
}

// ReverseOrder returns the reverse of the natural ordering, matching
// Comparator.reverseOrder() and Collections.reverseOrder().
func ReverseOrder[T cmp.Ordered]() Comparator[T] {
	return func(a, b T) int32 {
		return int32(cmp.Compare(b, a))
	}
}

// ComparatorComparing orders elements by an extracted sort key, matching
// Comparator.comparing / comparingInt / comparingLong / comparingDouble. The
// key type is a separate type parameter, so this is a free function rather than
// a method on Comparator.
func ComparatorComparing[T any, K cmp.Ordered](key func(T) K) Comparator[T] {
	return func(a, b T) int32 {
		return int32(cmp.Compare(key(a), key(b)))
	}
}

// ComparatorComparingReversed is Comparator.comparing(key).reversed(), provided
// directly because the transpiler can recognize the fused form.
func ComparatorComparingReversed[T any, K cmp.Ordered](key func(T) K) Comparator[T] {
	return func(a, b T) int32 {
		return int32(cmp.Compare(key(b), key(a)))
	}
}

// ComparatorNaturalOf returns the natural-order comparator for values that are
// only known to be Comparable at runtime, matching a raw Comparator.naturalOrder
// applied to generated reference types. It defers to the same CompareTo bridge
// the reflective array sort uses.
func ComparatorNaturalOf[T any]() Comparator[T] {
	return func(a, b T) int32 {
		return javaCompareValues(a, b)
	}
}

// Reversed returns a comparator imposing the reverse of this ordering, matching
// Comparator.reversed().
func (c Comparator[T]) Reversed() Comparator[T] {
	return func(a, b T) int32 {
		return c(b, a)
	}
}

// ThenComparing returns a comparator that falls back to next when this
// comparator reports equality, matching Comparator.thenComparing(Comparator).
func (c Comparator[T]) ThenComparing(next Comparator[T]) Comparator[T] {
	return func(a, b T) int32 {
		if result := c(a, b); result != 0 {
			return result
		}
		return next(a, b)
	}
}

// Compare applies the comparator, matching Comparator.compare.
func (c Comparator[T]) Compare(a, b T) int32 {
	return c(a, b)
}

// ComparatorThenComparingKey is Comparator.thenComparing(Function): it falls
// back to an extracted sort key when the receiver reports equality. The key type
// is a separate type parameter, so this cannot be a method.
func ComparatorThenComparingKey[T any, K cmp.Ordered](c Comparator[T], key func(T) K) Comparator[T] {
	return func(a, b T) int32 {
		if result := c(a, b); result != 0 {
			return result
		}
		return int32(cmp.Compare(key(a), key(b)))
	}
}

// SortWith sorts a list with an explicit comparator, matching
// Collections.sort(list, cmp) and List.sort(cmp). Java guarantees a stable sort
// for both, which is observable whenever the comparator examines only part of
// the element, so this uses a stable sort rather than the unstable sort that
// backs the natural-ordering path.
func SortWith[T any](l *List[T], c Comparator[T]) {
	if l == nil {
		panic(NewNullPointerException("Collections.sort on null"))
	}
	if c == nil {
		SortSliceStableNatural(l.elements)
		return
	}
	sort.SliceStable(l.elements, func(i, j int) bool {
		return c(l.elements[i], l.elements[j]) < 0
	})
}

// SortSliceWith sorts a slice with an explicit comparator, matching
// Arrays.sort(array, cmp). Like SortWith it is stable, as Java's reference-array
// sort is.
func SortSliceWith[T any](elements []T, c Comparator[T]) {
	if c == nil {
		SortSliceStableNatural(elements)
		return
	}
	sort.SliceStable(elements, func(i, j int) bool {
		return c(elements[i], elements[j]) < 0
	})
}

// SortSliceStableNatural stably sorts a slice by natural ordering, used when a
// comparator-taking Java overload is passed an explicit null comparator (which
// Java defines as "use natural ordering").
func SortSliceStableNatural[T any](elements []T) {
	sort.SliceStable(elements, func(i, j int) bool {
		return javaCompareValues(elements[i], elements[j]) < 0
	})
}

// MaxWith returns the largest element under an explicit comparator, matching
// Collections.max(coll, cmp). Java returns the first maximal element, so a later
// element replaces the incumbent only when it compares strictly greater.
func MaxWith[T any](l *List[T], c Comparator[T]) T {
	if l == nil || len(l.elements) == 0 {
		panic(NewNoSuchElementException("Collections.max on an empty collection"))
	}
	best := l.elements[0]
	for _, e := range l.elements[1:] {
		if c(e, best) > 0 {
			best = e
		}
	}
	return best
}

// MinWith returns the smallest element under an explicit comparator, matching
// Collections.min(coll, cmp). As with MaxWith, ties keep the earlier element.
func MinWith[T any](l *List[T], c Comparator[T]) T {
	if l == nil || len(l.elements) == 0 {
		panic(NewNoSuchElementException("Collections.min on an empty collection"))
	}
	best := l.elements[0]
	for _, e := range l.elements[1:] {
		if c(e, best) < 0 {
			best = e
		}
	}
	return best
}

// SortArrayWith is the comparator-taking Arrays.sort bridge for
// descriptor-bearing Java arrays. Java only offers this overload for reference
// arrays (a primitive array has no boxed element to hand a comparator), so a
// primitive array here means the source did not type-check as Java and the
// mismatch is reported rather than silently ignored.
//
// The element type comes from the comparator rather than the array, because a
// ReferenceArray erases its elements to `any`.
func SortArrayWith[T any](array any, c Comparator[T]) {
	if array == nil {
		panic(NewNullPointerException("Arrays.sort on null"))
	}
	values, ok := array.(*ReferenceArray)
	if !ok {
		panic(NewIllegalArgumentException("Arrays.sort with a comparator requires a reference array"))
	}
	if values == nil {
		panic(NewNullPointerException("Arrays.sort on null"))
	}
	if c == nil {
		SortSliceStableNatural(values.elements)
		return
	}
	sort.SliceStable(values.elements, func(i, j int) bool {
		left, leftOK := values.elements[i].(T)
		right, rightOK := values.elements[j].(T)
		if !leftOK || !rightOK {
			panic(NewClassCastException("Arrays.sort comparator does not accept the array's element type"))
		}
		return c(left, right) < 0
	})
}

// javaCompareValues compares two values by Java's natural ordering, returning
// the sign contract of Comparable.compareTo. It handles the primitive and String
// forms directly and otherwise bridges to a generated type's own CompareTo
// method.
//
// The bridge is reflective because a Java class implementing Comparable<Foo>
// generates `CompareTo(other *Foo) int32` — the parameter is the concrete type,
// not `any`, so no single Go interface can capture every such method.
func javaCompareValues(left, right any) int32 {
	switch left := left.(type) {
	case string:
		if right, ok := right.(string); ok {
			return int32(cmp.Compare(left, right))
		}
	case int8:
		if right, ok := right.(int8); ok {
			return int32(cmp.Compare(left, right))
		}
	case int16:
		if right, ok := right.(int16); ok {
			return int32(cmp.Compare(left, right))
		}
	case int32:
		if right, ok := right.(int32); ok {
			return int32(cmp.Compare(left, right))
		}
	case int64:
		if right, ok := right.(int64); ok {
			return int32(cmp.Compare(left, right))
		}
	case float32:
		if right, ok := right.(float32); ok {
			return int32(cmp.Compare(left, right))
		}
	case float64:
		if right, ok := right.(float64); ok {
			return int32(cmp.Compare(left, right))
		}
	case bool:
		if right, ok := right.(bool); ok {
			switch {
			case left == right:
				return 0
			case left:
				return 1
			default:
				return -1
			}
		}
	}
	if result, ok := compareViaCompareTo(left, right); ok {
		return result
	}
	// A named type over a Go ordered kind (or an unsigned width the switch above
	// does not spell) still has a natural ordering. This is checked after the
	// CompareTo bridge so a type that defines its own ordering wins over its
	// underlying representation.
	if result, ok := compareViaReflectOrdering(left, right); ok {
		return result
	}
	panic(NewClassCastException("value is not naturally comparable"))
}

// compareViaReflectOrdering compares two same-typed values whose Go kind has a
// natural ordering, and reports whether it could.
func compareViaReflectOrdering(left, right any) (int32, bool) {
	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	if !leftValue.IsValid() || !rightValue.IsValid() || leftValue.Type() != rightValue.Type() {
		return 0, false
	}
	switch leftValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int32(cmp.Compare(leftValue.Int(), rightValue.Int())), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int32(cmp.Compare(leftValue.Uint(), rightValue.Uint())), true
	case reflect.Float32, reflect.Float64:
		return int32(cmp.Compare(leftValue.Float(), rightValue.Float())), true
	case reflect.String:
		return int32(cmp.Compare(leftValue.String(), rightValue.String())), true
	}
	return 0, false
}

// compareViaCompareTo invokes a generated `CompareTo` method on left, passing
// right, and reports whether such a method was found and applicable.
func compareViaCompareTo(left, right any) (int32, bool) {
	if left == nil {
		return 0, false
	}
	receiver := reflect.ValueOf(left)
	if !receiver.IsValid() {
		return 0, false
	}
	if (receiver.Kind() == reflect.Pointer || receiver.Kind() == reflect.Interface) && receiver.IsNil() {
		panic(NewNullPointerException("compareTo on null"))
	}
	method := receiver.MethodByName("CompareTo")
	if !method.IsValid() {
		return 0, false
	}
	methodType := method.Type()
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 {
		return 0, false
	}
	switch methodType.Out(0).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	default:
		return 0, false
	}
	argument := reflect.ValueOf(right)
	if !argument.IsValid() {
		if methodType.In(0).Kind() != reflect.Pointer && methodType.In(0).Kind() != reflect.Interface {
			return 0, false
		}
		argument = reflect.Zero(methodType.In(0))
	}
	if !argument.Type().AssignableTo(methodType.In(0)) {
		return 0, false
	}
	return int32(method.Call([]reflect.Value{argument})[0].Int()), true
}
