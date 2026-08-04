# Raw unbound specialized-bridge TDD fixture

This frozen whole-application target exercises an unbound method reference to
a raw generic receiver whose runtime object may be either the generic base or a
specialized subclass. Java accepts erased-bound arguments at the raw functional
interface. A plain base receiver stores a differently typed implementation of
the bound, while a specialized receiver dispatches through javac's synthetic
erased bridge.

The specialized bridge must cast its erased argument to `First` before entering
the override body. The final rejected call therefore throws
`ClassCastException` without incrementing either body counter. The oracle also
proves that the successful specialized call dispatches to the override and then
to the base body, while the plain raw call accepts and stores `Second`.

The fixture is deterministic and has no external inputs or dependencies. It is
a passing parity target: generated Go must reproduce the exact frozen Java
output. The Java source and oracle remain unchanged from the original known-gap
target.
