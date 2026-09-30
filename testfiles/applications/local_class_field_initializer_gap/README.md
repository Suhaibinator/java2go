# Local-class field-initializer TDD fixture

This fixture isolates declaration-order field initialization in a method-local
class. Two fields call a side-effecting enclosing helper, the second initializer
reads the first field, and the constructed object is observed through both
direct field access and an instance method. Markers before and after construction
make skipped or reordered effects visible in the output.

The local-class lowering emits an ordered initializer method on the synthetic
Go type. Every allocation first installs captured values, then executes each
declared instance-field initializer through the synthetic receiver. Recursive
allocations made from an initializer or instance method forward those captures
through the active receiver. The fixture is a passing regression test for that
behavior; explicit local-class constructors remain a separate TDD gap.

The program is deterministic and has no external inputs or dependencies.
