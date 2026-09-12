# Generic and covariant dispatch applications

This fixture runs six small applications across seven Java packages. The live
JDK oracle and the generated Go executable must print the same six lines:

- Generic method calls through a base reference, an interface reference, and
  an inherited method all select the override (`base:interface:indirect:222`).
- An unchecked generic result is discarded without a checkcast; a consumed
  result fails after body effects. Both base and interface null receivers
  evaluate arguments before throwing; a structurally matching class fails the
  nominal interface result check (`cast:null:null:nominal:232334`).
- Number-bounded methods dispatch through abstract classes and interfaces,
  preserving Integer and Long results (`7:9:16`).
- A generic specialization accepts an erased interface argument, checks the
  narrow argument before the override, and widens its concrete result without
  checking or dereferencing null (`13:20:cast:true`).
- An ordinary covariant factory dispatches through base, child, interface, and
  bound method-reference calls. Superclass return views preserve allocation
  identity, default hash codes, and null (`2:true:true:60`).

- A generic call consumed as an interface checks that interface, preserving a
  valid result whose concrete class differs from the argument-inferred class (`2`).

These applications were written and run with javac/java before implementation.
The original transpiler failed the generic application with undeclared `T`
interface signatures and a missing interface helper constructor. The concrete
result bridge fixture failed because its raw call required the specialized
argument type. The ordinary factory failed because its interface fallback
returned a Child pointer where Go required a Base pointer.

## Current boundaries

The new virtual generic entrypoints cover bare method type parameters with
Object erasure or a single non-parameterized upper bound. Nested parameterized
uses, generic varargs, intersection bounds, general wildcard variance, and
concrete-class-bound generic subtyping retain their existing limitations.
Ordinary covariance supports source-backed concrete class result hierarchies;
multiple incompatible ancestor bridge descriptors remain conservatively gated.
Generated Java calls retain narrow results through the exact hidden method;
the public Go wrapper of an ordinary covariant override exposes its ancestor
result type so it also satisfies inherited Go interfaces.
