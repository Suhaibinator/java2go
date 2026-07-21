# Static method-hiding TDD fixture

Java static methods belong to their declaring classes, so a child may hide a
same-signature parent method and class-qualified calls select different bodies.
Generated static methods currently become package-level Go functions without a
declaring-class namespace. The resolver assigns colliding static methods unique
package-level names while each Java call remains bound to its declaring class.

The fixture entered the corpus as a generated-Go compilation gap caused by a
duplicate `kind` declaration. It now passes with the exact Java oracle
`parent:child`. It is deterministic and has no external inputs or dependencies.
