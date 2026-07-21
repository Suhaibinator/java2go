# Try/finally loop-control parity

Java routes `continue` and `break` through the active `finally` block and then
resumes the surrounding loop transfer. This fixture proves that the generated
Go executes the same `t0f0t1f1e` trace byte-for-byte: both transfers leave the
try body, `finally` runs, and only then does loop control resume.

The lowering records transfers that cross a generated try/catch/finally closure
and replays them at the nearest Go scope containing their Java target. Transfers
to loops or switches inside the closure remain ordinary Go branches.
