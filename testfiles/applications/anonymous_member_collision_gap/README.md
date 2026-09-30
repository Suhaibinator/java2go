# Anonymous member-collision TDD fixture

Java keeps fields and methods in separate member namespaces. With local-variable
type inference, the exact anonymous type remains available inside the declaring
method, so Java permits direct access to both a field named `score` and a method
named `score()` on the same anonymous object.

Historically, anonymous-class lowering preserved both selector spellings on one
Go type, which Go rejected because fields and methods share a selector
namespace. The fixture now passes and guards the distinct Java member
resolution at both the field access and method call, including retention of the
anonymous value's exact synthetic type.

The program is deterministic and has no external inputs or dependencies.
