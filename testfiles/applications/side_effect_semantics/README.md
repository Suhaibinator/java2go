# Side-effect and evaluation-order parity

This fixture is an executable Java oracle for expression semantics where a
translation can return the right value while still performing observable work
in the wrong order.

The output lines cover these invariants:

- `SHORT`: `&&` and `||` skip the unselected right operand and retain left-to-right effects.
- `TERNARY`: only the selected branch runs, including for nested conditionals.
- `INSTANCE`: the receiver runs first, arguments run left to right, and the method body runs last.
- `NULL_INSTANCE`: arguments run before the null check, but a method body must never execute on a null Java receiver. This was a confirmed java2go divergence.
- `STATIC_QUALIFIER`: Java still evaluates the expression that qualifies a static call before its arguments, even though dispatch discards the value. This was a confirmed java2go divergence.
- `COMPOUND`: a compound assignment captures the old target before evaluating its mutating right-hand side.
- `FINALLY`: a return expression is evaluated before `finally`, while `finally` still runs before control returns.
- `RECURSION`: effects in recursive arguments and unwind steps retain their Java order.

Next TDD additions are intentionally kept in separate known-gap fixtures until
their lowerings are fixed: `break`/`continue` through `try/finally`, array-index
mutation in assignment targets, and the different exception timing of simple
versus compound array assignment.
