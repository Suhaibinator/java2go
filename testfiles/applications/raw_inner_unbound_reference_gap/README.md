# Raw inner unbound method-reference TDD fixture

This frozen whole-application target assigns one unbound method reference for a
raw generic member class and invokes it with two different concrete outer/inner
type combinations. Java erases the raw receiver slot while preserving the
distinct `OuterBound` and `InnerBound` bounds accepted by the target method.

Both calls deliberately heap-pollute their typed owners through that raw view.
An explicit raw outer alias and the raw inner alias keep the immediate reads
valid through their erased bounds, while later typed reads throw
`ClassCastException`. The oracle therefore covers receiver adaptation, separate
outer and inner erasures, mutation, delayed casts, and one raw reference
accepting multiple invariant instantiations.

The fixture is deterministic and has no external inputs or dependencies. Its
known-gap manifest pins the earliest current generated-Go compilation failure.
It must be promoted to `passing` only when generated Go reproduces the exact
Java output; the Java source and oracle must not be weakened to make it green.
