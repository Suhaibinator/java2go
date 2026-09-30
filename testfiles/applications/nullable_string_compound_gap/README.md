# Nullable String compound-assignment TDD fixture

Java `String +=` is concatenation, so it applies String conversion to both
operands. A null left operand therefore contributes the four characters
`null`; it is not dereferenced.

This application exercises that rule for both a null-initialized local and an
implicitly null instance field. A Java `try`/`catch` around the local operation
made the former generated null panic deterministic and allowed the field case
to run too. The fixture now passes and guards the exact `nullx|nullx` oracle,
including null conversion for both local and default field storage.
