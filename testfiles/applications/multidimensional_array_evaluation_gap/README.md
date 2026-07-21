# Multidimensional-array evaluation-order TDD fixture

This fixture isolates the ordering rules for a Java array creation expression
with several explicit dimensions. Each dimension calls a recursive helper that
records a distinct marker before returning its size. The second size is
negative, while the third expression still has an observable side effect.

Java evaluates all three expressions from left to right, recording `123`, and
only then performs the negative-size check. Catching
`NegativeArraySizeException` records `8`, so the exact oracle output is `1238`.

The original generated Go allocation built one nesting level at a time. It
recorded `12`, panicked while allocating the negative second dimension, and
skipped the third expression, producing `128`. The pinned output mismatch is a
TDD target for staging all explicit dimensions before allocation.

The dimensions are tiny and allocation never begins in the negative case, so
the fixture has a negligible and deterministic memory footprint.
