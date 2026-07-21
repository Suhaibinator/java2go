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

Generated Go represents the value as a `[]*Base` without the original
`Child[]` component metadata. It therefore accepts the replacement, records
the normally unreachable `x`, changes the first value from `4` to `9`, and
changes the second recursive sum from `9` to `14`. The fixture pins that exact
output mismatch as a TDD target for a future reified-array representation or
checked reference-array store.

The program is deterministic and has no external inputs or dependencies.
