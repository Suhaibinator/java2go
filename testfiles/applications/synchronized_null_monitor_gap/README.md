# Null synchronized-monitor parity

Java evaluates the expression in `synchronized (expression)` exactly once. If
that expression produces `null`, monitor entry throws `NullPointerException`
before any statement in the body executes.

The trace makes each phase observable: `1` is appended by the lock expression,
`2` would be appended by the body, `3` would follow normal monitor completion,
`4` is appended by the Java `NullPointerException` handler, and `5` follows the
try/catch. Java therefore prints `TRACE=145`.

Generated Go now rejects both an untyped `nil` interface and interface-wrapped
typed nil references at monitor entry. It raises the modeled Java
`NullPointerException` after evaluating the lock expression and before executing
the synchronized body, so the generated program also prints `TRACE=145`.
