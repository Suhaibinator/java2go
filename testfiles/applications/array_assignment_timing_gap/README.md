# Array-assignment exception timing known gap

For a simple Java array assignment, evaluation proceeds through the array
expression, index, and right-hand side before the store performs its null/bounds
check. Consequently this fixture records `i`, then `r`, and catches a
`NullPointerException` as `c`.

The generated representation is a Go slice. Indexing its nil value currently
raises a native bounds panic, which normalization classifies as
`ArrayIndexOutOfBoundsException`; the Java `NullPointerException` catch therefore
does not handle it. This fixture remains `known_gap` until array-store lowering
stages the operands and performs Java's checks at the correct point.

`JAVA2GO_PARITY_STRICT=1` promotes the recorded gap to a failing TDD target.
