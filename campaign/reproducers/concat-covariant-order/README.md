# Earlier field read versus later covariant invocation in concatenation

This valid-Java case was discovered while testing inherited-overload dispatch.
JDK 21 evaluates the left `counter.count` before `view.touch` mutates that field;
its output is `0:true`. A generated formatting call must preserve that snapshot
when the later operand includes the covariant reference-result projection.

This is an independent pending compiler regression, not accepted campaign parity.

Observed generated Go output: `10:true` (diagnostic test driver invoking the
unchanged generated `Main`, 2026-09-28). The first field operand is passed to a
flattened `fmt.Sprintf` alongside a later call, without an earlier snapshot.
