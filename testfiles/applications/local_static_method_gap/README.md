# Local static-method TDD fixture

Modern Java permits a method-local class to declare static members. This
fixture calls an overloaded static method through the local class name and uses
a nested static helper call, exercising declaration, overload resolution, and
type-qualified invocation without constructing an instance.

The synthetic local-class lowering registers the hoisted class scope before it
renders member bodies, emits static members as uniquely named package
functions, and resolves type-qualified calls through the synthetic scope. The
fixture is a passing regression test for that behavior.

The program is deterministic and has no external inputs or dependencies.
