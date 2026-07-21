# Labeled non-loop break known gap

Java permits `break outer` when `outer` labels an arbitrary statement, including
a block. Go only permits a labeled `break` when the label identifies a `for`,
`switch`, or `select`, so replaying this transfer after the generated
try/finally closure is not sufficient on its own.

This fixture deliberately remains `known_gap`; it keeps the broader Java
control-transfer mismatch visible without weakening the now-supported loop
label lowering. Strict parity mode treats it as a failing TDD target.
