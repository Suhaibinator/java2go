# Synchronized loop-control known gap

Java permits `continue` from a `synchronized` block to an enclosing loop and
releases the monitor before transferring control. The current synchronized
lowering uses a generated Go function literal so monitor exit can be deferred;
a native Go `continue` inside that function cannot target the surrounding loop.

The oracle also records work performed inside a normally completing inner
`finally`. This makes the expected Java path deterministic: the first two loop
iterations continue after synchronized cleanup, while the third completes the
loop body and prints `1782783789`.

This fixture remains `known_gap` until synchronized abrupt completions are
propagated across the generated closure with monitor release preserved.
