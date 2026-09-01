# Covariant array-store TDD fixture

This fixture isolates Java's reified reference-array component type while also
making initialization, recursion, aliasing, and evaluation order observable.
Two `Child` objects are constructed left to right in a `Child[]`, which is then
used through a covariant `Base[]` reference. A recursive traversal records the
array's values before and after an adversarial store.

The store expression `select(view)[index()] = replacement()` records `a`, `i`,
`r`, and the replacement object's constructor marker `b`. Java evaluates all
of those effects before its runtime component-type check rejects the `Base`
value with `ArrayStoreException`. The catch records `e`, the original `Child`
remains at index zero, and both recursive sums stay `9`.

This formerly failed when generated Go lost the original `Child[]` component
metadata, accepted the replacement, recorded the normally unreachable `x`, and
changed the second recursive sum from `9` to `14`. The fixture now passes and
guards the reified component check, the exception point after all operand side
effects, and preservation of the original array contents.

The program is deterministic and has no external inputs or dependencies.
