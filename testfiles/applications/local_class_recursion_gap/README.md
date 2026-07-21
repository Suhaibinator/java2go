# Local-class recursion TDD fixture

This fixture declares a method-local class that captures an effectively final
parameter and recursively invokes its own instance method. Java prints `11`.
The fixture entered the corpus when the hoisted type emitted the recursive call
as bare `down()` and generated Go did not compile. Synthesized methods now carry
their Java method identity, return type, parameters, and complete class method
table, so the same application matches Java byte for byte.

The fixture is deterministic and has no external inputs or dependencies.
