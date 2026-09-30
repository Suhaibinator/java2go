# Reflection argument adaptation

This application checks empty/null Class[] and Object[] reflection argument
arrays, distinguishes one `(Object)null` argument from a null varargs array,
and verifies that a throwing constructor retains its invocation target cause.
The binary-name call deliberately includes a comment before the argument list.

OpenJDK 21 produced `plugin/plugin/arity/constructor` before argument adaptation.
Generated Go initially failed because a `*stdjava.ReferenceArray` could not be
passed as `*stdjava.Class`. This fixture now checks the complete application
through the Java/Go parity harness.
