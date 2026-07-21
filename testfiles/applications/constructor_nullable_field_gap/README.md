# Constructor nullable-field TDD fixture

This fixture isolates Java's interaction between constructor-time virtual
dispatch and the default value of a reference field. A superclass instance
initializer calls a child override before the child's `String` initializer has
run. Java therefore passes `null` to `String.valueOf` and prints `null:ready`.

The current generated representation stores ordinary Java `String` fields as
Go `string` values. Before initialization that storage contains `""`, so the
generated application prints `:ready`. The fixture is intentionally pinned as
an output-stage `known_gap`, giving a byte-exact TDD target for a future nullable
field representation without broadening the smaller method-resolution fix in
the recursive object-model fixture.

The program is deterministic and has no external inputs or dependencies.
