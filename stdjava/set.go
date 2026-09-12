package stdjava

import "strings"

// Set shares Map's Java equality/hash and sorted-key semantics. Hash sets keep
// encounter order; tree sets use comparison equality and sorted iteration.
type Set[T any] struct{ backing *Map[T, bool] }

func NewSet[T any]() *Set[T]     { return &Set[T]{backing: NewMap[T, bool]()} }
func NewTreeSet[T any]() *Set[T] { return NewTreeSetWith[T](nil) }
func NewTreeSetWith[T any](comparator Comparator[T]) *Set[T] {
	return &Set[T]{backing: NewTreeMapWith[T, bool](comparator)}
}
func (s *Set[T]) Add(element T, execution ...*Execution) bool {
	return !s.backing.Put(element, true, execution...)
}
func (s *Set[T]) Contains(element any, execution ...*Execution) bool {
	return s.backing.ContainsKey(element, execution...)
}
func (s *Set[T]) Remove(element any, execution ...*Execution) bool {
	return s.backing.Remove(element, execution...)
}
func (s *Set[T]) Size() int32   { return s.backing.Size() }
func (s *Set[T]) IsEmpty() bool { return s.backing.IsEmpty() }
func (s *Set[T]) Clear()        { s.backing.Clear() }
func (s *Set[T]) Slice() []T    { return s.backing.KeySet() }
func (s *Set[T]) String() string {
	elements := s.Slice()
	parts := make([]string, len(elements))
	for i, element := range elements {
		parts[i] = StringValueOf(element)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
