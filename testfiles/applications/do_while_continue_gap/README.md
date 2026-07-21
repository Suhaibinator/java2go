# Do-while continue known gap

A Java `continue` in a do-while loop jumps to the condition check. The current
lowering appends that check to a Go `for` body, so a native Go `continue` skips
it and starts another iteration. Routing the transfer through `finally` does not
change that underlying target mismatch.

This fixture remains `known_gap` until do-while lowering gives continues an
explicit condition-check target. Other loop kinds remain covered by passing
runtime parity tests.
