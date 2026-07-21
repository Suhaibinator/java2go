# Nullable String compound-assignment TDD fixture

Java `String +=` is concatenation, so it applies String conversion to both
operands. A null left operand therefore contributes the four characters
`null`; it is not dereferenced.

This application exercises that rule for both a null-initialized local and an
implicitly null instance field. A Java `try`/`catch` around the local operation
turns the generated null panic into deterministic output and lets the field case
run too. The Java oracle is `nullx|nullx`; current generated Go reports the
caught local failure and treats the field's empty Go zero value as text.
