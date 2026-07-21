# Constructor nullable-field TDD fixture

This fixture isolates Java's interaction between constructor-time virtual
dispatch and the default value of a reference field. Both a superclass field
initializer and the superclass constructor call a child override before the
child's `String` initializer has run. A helper class reads that child field both
directly and through a `String`-returning method, so the fixture also proves that
the state is visible consistently across class and method-ABI boundaries. Java
passes `null` to `String.valueOf` in both early calls, then observes `ready` from
the subsequent child initializer and constructor body.

The current generated representation stores ordinary Java `String` fields as
Go `string` values. Before initialization that storage contains `""`, so the
generated application prints `:ready`. The fixture is intentionally pinned as
an output-stage `known_gap`, giving a byte-exact TDD target for a future nullable
field representation. Keeping the later child-initializer and constructor
observations in the same oracle also prevents a fix from delaying initialization
too far or reordering the field initializer.

The program is deterministic and has no external inputs or dependencies.
