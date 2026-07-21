# Anonymous member-collision TDD fixture

Java keeps fields and methods in separate member namespaces. With local-variable
type inference, the exact anonymous type remains available inside the declaring
method, so Java permits direct access to both a field named `score` and a method
named `score()` on the same anonymous object.

The current anonymous-class lowering preserves both selector spellings on one
Go type, which Go rejects. Unlike method-local named classes, anonymous values
are not yet associated with their synthetic class scope at later call and field
access sites, so simply renaming one declaration would not retarget both uses.
This fixture pins that distinct resolution gap.

The program is deterministic and has no external inputs or dependencies.
