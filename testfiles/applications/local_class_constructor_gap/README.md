# Local-class constructor TDD fixture

This fixture isolates constructor semantics for a method-local class. Two
side-effecting arguments establish left-to-right evaluation, the constructor
records its own effect, and its body combines the arguments into an instance
field that is later read through a method.

This formerly failed when local-class lowering ignored constructor declarations
and dropped the Java argument list, so neither argument effects nor the
constructor body executed. The fixture now passes and guards left-to-right
argument evaluation, constructor effects, field assignment, and the later
method read separately from default field initialization and member naming.

The program is deterministic and has no external inputs or dependencies.
