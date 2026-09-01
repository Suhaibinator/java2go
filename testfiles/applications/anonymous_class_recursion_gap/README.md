# Anonymous-class recursion TDD fixture

This fixture creates an anonymous subclass whose override recursively computes
a factorial. Java prints `720`. The fixture entered the corpus when the
generated anonymous type emitted the recursive call as bare `apply()` and did
not compile. Synthesized method metadata now resolves that call through the
anonymous receiver and preserves the method's `int` result type, producing
byte-exact parity.

It also infers an anonymous class with `var`, nulls that receiver, and verifies
that a side-effecting argument runs before `NullPointerException` without
allowing the synthetic method body to execute on a nil Go receiver.

The fixture is deterministic and has no external inputs or dependencies.
