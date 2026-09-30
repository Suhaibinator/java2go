# Do-while continue parity

A Java `continue` in a do-while loop jumps to the condition check. The current
lowering gives that phase an explicit generated label, so ordinary and labeled
continues evaluate the condition before deciding whether to iterate again.

This fixture verifies byte-exact parity when the transfer crosses `finally`.
Focused runtime tests additionally cover condition side effects, labeled
continues, nested loops, break, and return.
