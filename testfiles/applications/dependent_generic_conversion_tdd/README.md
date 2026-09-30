# Dependent generic conversion TDD fixture

This frozen parity target exercises a type parameter whose upper bound is a
second type parameter while the concrete value is a Java subclass. It combines
nested generic-call inference, both explicit and implicit concrete dependent
views, and a generic constructor that widens `T` to `B` in its body.

Each route uses distinct values and contributes to an order-sensitive checksum,
so a dropped superclass view or an incorrect inference shortcut cannot silently
produce the oracle. The manifest deliberately starts as `passing`: generated-Go
failure is an active TDD target. Runtime and live memory are trivial.
