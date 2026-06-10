package stdjava

// Optional models java.util.Optional<T>: a container that may or may not hold a
// non-null value. A nil internal pointer represents Optional.empty().
type Optional[T any] struct {
	value *T
}

// OptionalOf returns an Optional describing the given value, matching
// Optional.of. Java throws on a null argument; that null-check is not modelled.
func OptionalOf[T any](value T) Optional[T] {
	return Optional[T]{value: &value}
}

// OptionalEmpty returns an empty Optional, matching Optional.empty.
func OptionalEmpty[T any]() Optional[T] {
	return Optional[T]{}
}

// OptionalOfNullable returns an empty Optional if present is false, otherwise an
// Optional of value, matching Optional.ofNullable. Because Go non-pointer values
// are not nullable, callers pass an explicit presence flag.
func OptionalOfNullable[T any](value T, present bool) Optional[T] {
	if !present {
		return Optional[T]{}
	}
	return Optional[T]{value: &value}
}

// Some reports whether a value is present.
func (o Optional[T]) Some() bool {
	return o.value != nil
}

// IsPresent reports whether a value is present, matching Optional.isPresent.
func (o Optional[T]) IsPresent() bool {
	return o.value != nil
}

// IsEmpty reports whether no value is present, matching Optional.isEmpty.
func (o Optional[T]) IsEmpty() bool {
	return o.value == nil
}

// Get returns the contained value, matching Optional.get. It panics when empty,
// preserving Java's NoSuchElementException contract.
func (o Optional[T]) Get() T {
	if o.value == nil {
		panic("Optional.get() called on an empty Optional")
	}
	return *o.value
}

// OrElse returns the value if present, otherwise other, matching
// Optional.orElse.
func (o Optional[T]) OrElse(other T) T {
	if o.value == nil {
		return other
	}
	return *o.value
}

// IfPresent invokes action with the value if present, matching
// Optional.ifPresent.
func (o Optional[T]) IfPresent(action func(T)) {
	if o.value != nil {
		action(*o.value)
	}
}

// OptionalMap applies mapper to the contained value if present and returns an
// Optional of the result, matching Optional.map. It is a free function because
// Go methods cannot introduce a new type parameter for the result.
func OptionalMap[T, R any](o Optional[T], mapper func(T) R) Optional[R] {
	if o.value == nil {
		return Optional[R]{}
	}
	return OptionalOf(mapper(*o.value))
}
