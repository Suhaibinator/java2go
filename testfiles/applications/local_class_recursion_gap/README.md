# Local-class recursion TDD fixture

This fixture declares a method-local class that captures an effectively final
parameter and recursively invokes its own instance method. Java prints `11`.
The hoisted generated type currently emits the recursive call as bare `down()`
instead of selecting the synthetic local-class receiver, so generated Go does
not compile.

The fixture is deterministic and has no external inputs or dependencies.
