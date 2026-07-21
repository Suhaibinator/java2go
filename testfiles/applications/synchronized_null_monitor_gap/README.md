# Null synchronized-monitor known gap

Java evaluates the expression in `synchronized (expression)` exactly once. If
that expression produces `null`, monitor entry throws `NullPointerException`
before any statement in the body executes.

The trace makes each phase observable: `1` is appended by the lock expression,
`2` would be appended by the body, `3` would follow normal monitor completion,
`4` is appended by the Java `NullPointerException` handler, and `5` follows the
try/catch. Java therefore prints `TRACE=145`.

Generated Go currently accepts `nil` as a monitor-map key, executes the body,
and completes normally, printing `TRACE=1235`. The fixture pins that exact
output mismatch until monitor entry rejects null after evaluating the lock
expression and before executing the synchronized body.
