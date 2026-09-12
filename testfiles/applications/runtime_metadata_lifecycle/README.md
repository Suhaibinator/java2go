# Runtime metadata lifecycle

This application checks class-literal laziness, `Class.forName` initialization,
constructor execution, hidden superclass fields, primitive boxing and widening,
nullable String/reference fields, reference identity, virtual synchronized
invocation, target exception causes, lookup errors, and final-field protection.

OpenJDK 21 produced `expected.stdout` before the lifecycle fixes. The first Go
attempt failed compilation on `InvocationTargetException.getCause`. Once that
was lowered, execution exposed superclass-view identity and virtual-dispatch
mismatches (`false/base` instead of `true/plugin`). The final application is a
passing fixture in the Java/Go parity harness.
