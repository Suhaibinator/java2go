# Anonymous-class recursion TDD fixture

This fixture creates an anonymous subclass whose override recursively computes
a factorial. Java prints `720`. The generated anonymous type currently emits
the recursive call as bare `apply()` instead of dispatching through the
synthesized receiver, so generated Go does not compile.

The fixture is deterministic and has no external inputs or dependencies.
