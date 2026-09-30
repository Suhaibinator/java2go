package stdjava

// EvaluationValue snapshots a Java expression before subsequent call arguments
// are evaluated. Go orders calls left to right, but may otherwise postpone a
// plain variable read until a later argument's call has modified that variable.
func EvaluationValue[T any](value T) T { return value }
