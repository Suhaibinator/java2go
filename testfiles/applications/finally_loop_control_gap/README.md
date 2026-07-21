# Try/finally loop-control known gap

Java routes `continue` and `break` through the active `finally` block and then
resumes the surrounding loop transfer. The current try/finally lowering places
those statements in a generated Go closure, where they are not lexically inside
the loop and therefore do not compile.

This fixture intentionally remains `known_gap`. The ordinary application parity
suite must reproduce the exact compiler failure. Running with
`JAVA2GO_PARITY_STRICT=1` turns it into a failing TDD target until the lowering
is corrected and the fixture can be promoted to `passing`.
