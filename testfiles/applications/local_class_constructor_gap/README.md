# Local-class constructor TDD fixture

This fixture isolates constructor semantics for a method-local class. Two
side-effecting arguments establish left-to-right evaluation, the constructor
records its own effect, and its body combines the arguments into an instance
field that is later read through a method.

The current local-class lowering hoists fields and ordinary methods but ignores
constructor declarations. Construction also drops the Java argument list, so
neither argument effects nor the constructor body execute. The fixture pins
that behavior separately from default field initialization and member naming.

The program is deterministic and has no external inputs or dependencies.
