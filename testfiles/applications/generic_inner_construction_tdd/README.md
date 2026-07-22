# Generic inner construction TDD fixture

This frozen parity target combines three generic member-class paths in one
small deterministic application: a raw generic outer/inner pair whose method
mutates both objects, unqualified construction of an inherited generic inner
class, and explicitly qualified construction of that same inherited class.

The output records state before and after each mutation, the final enclosing
objects, and an order-sensitive checksum. The manifest deliberately starts as
`passing`: any generated-Go failure is transpiler work, not a tolerated known
gap. The fixture uses no external input, nondeterminism, or material memory.
