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
skipped the third expression, producing `128`. The fixture entered the suite as
a pinned known gap.

Generated multidimensional allocations now pass all explicit dimensions as
arguments to a synthetic allocator function. Go evaluates those arguments once
and left to right; the function then checks all staged values before allocating
any level and raises the modelled `NegativeArraySizeException` when needed. The
fixture is therefore promoted to passing byte-exact parity.

The dimensions are tiny and allocation never begins in the negative case, so
the fixture has a negligible and deterministic memory footprint.

The oracle deliberately observes exception identity and timing rather than the
detail returned by `getMessage()`. Java specifies `NegativeArraySizeException`
for this case but does not specify that implementation-dependent message;
synthetic multidimensional checks therefore use a stable empty detail string.
