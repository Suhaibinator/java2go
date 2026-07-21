# Local-class field-initializer TDD fixture

This fixture isolates declaration-order field initialization in a method-local
class. Two fields call a side-effecting enclosing helper, the second initializer
reads the first field, and the constructed object is observed through both
direct field access and an instance method. Markers before and after construction
make skipped or reordered effects visible in the output.

The current local-class lowering hoists field declarations but constructs the
synthetic Go value without running their Java initializer expressions. The
fixture pins that semantic mismatch independently from field/method naming and
local-class constructor support.

The program is deterministic and has no external inputs or dependencies.
