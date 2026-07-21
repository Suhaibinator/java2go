# Synchronized loop-control parity

Java permits `continue` from a `synchronized` block to an enclosing loop and
releases the monitor before transferring control. The generated Go records the
abrupt completion outside its monitor function literal, returns through the
deferred monitor exit, and then replays the transfer in the enclosing loop.

The oracle also records work performed inside a normally completing inner
`finally`. This makes the expected Java path deterministic: the first two loop
iterations continue after synchronized cleanup, while the third completes the
loop body and prints `1782783789`.

Focused transpiler coverage also exercises break, labeled transfers, returns,
throws, nested monitors, do-while continuation, and competing `finally`
completion while requiring each released monitor to be immediately reusable.
