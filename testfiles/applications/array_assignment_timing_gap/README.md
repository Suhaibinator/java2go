# Array-assignment exception timing parity

For a simple Java array assignment, evaluation proceeds through the array
expression, index, and right-hand side before the store performs its null/bounds
check. Consequently this fixture records `i`, then `r`, and catches a
`NullPointerException` as `c`.

Generated simple array stores use a staged runtime helper. Go evaluates the
array expression, index, and value arguments from left to right; the helper then
performs Java's null and bounds checks before storing. This preserves both the
observable `ir` side effects and the caught `NullPointerException` marker `c`.

The fixture is passing and pins byte-exact Java/Go output parity.
