# Synthetic member-collision TDD fixture

Java keeps fields and methods in separate member namespaces, so a local class
may declare both a field named `value` and a zero-argument method named
`value()`. The fixture explicitly assigns and reads the field, then invokes the
same-spelled method through an instance of the local class.

The generated Go synthetic struct gives the method a collision-safe name and
uses its registered local-class scope to retarget method calls while preserving
field selectors. The fixture is a passing application-level regression test.

The program is deterministic and has no external inputs or dependencies.
