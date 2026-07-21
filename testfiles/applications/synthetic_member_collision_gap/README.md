# Synthetic member-collision TDD fixture

Java keeps fields and methods in separate member namespaces, so a local class
may declare both a field named `value` and a zero-argument method named
`value()`. The method reads the same-spelled field and is invoked through an
instance of the local class.

The generated Go synthetic struct currently preserves both spellings. Go keeps
fields and methods in one selector namespace and rejects the generated type at
compile time. This fixture pins that mismatch as a strict application-level TDD
target.

The program is deterministic and has no external inputs or dependencies.
