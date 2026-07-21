# Recursive object-model parity fixture

This fixture is a deterministic object-oriented expression and rule engine. It
constructs a recursive expression tree, evaluates it through both concrete and
interface references, exercises constructor-time virtual dispatch, traverses a
generic recursive chain, and verifies several Java class-selection rules.

## Coverage

- Recursive interface-based expression evaluation and inherited interface
  default methods that call abstract methods through the concrete receiver.
- A superclass instance-field initializer that invokes a child override before
  child fields are initialized, observing Java's required zero value.
- Recursive base-class methods whose inner virtual calls dispatch to a child,
  plus an explicit `super` entry point.
- Direct and mutual recursion, recursive generic object references, and null
  termination.
- A non-static inner class that captures its enclosing object and recursively
  invokes itself.
- Instance-field hiding through base and child methods.
- Cross-package construction, calls, casts, and interface references.

The fixture entered the corpus as a pinned `known_gap`: a direct call to an
inherited default method through a concrete class was emitted with unresolved
Java casing and failed generated-Go compilation. It is promoted to `passing`
only after that same application compiles and its output matches Java exactly.

Follow-up adversarial targets discovered alongside this fixture include static
method hiding and recursive methods declared in local or anonymous classes.
Those remain separate so each future failure has one precise TDD signal.

The application has no stdin, files, clocks, randomness, network access,
locale-sensitive formatting, or environment-dependent behavior.
