# Static method-hiding TDD fixture

Java static methods belong to their declaring classes, so a child may hide a
same-signature parent method and class-qualified calls select different bodies.
Generated static methods currently become package-level Go functions without a
declaring-class namespace, producing a duplicate `kind` declaration.

The fixture pins that generated-Go compilation failure while retaining the
exact Java oracle `parent:child`. It is deterministic and has no external
inputs or dependencies.
