# Package-private class and local identifier collision

This independent valid-Java reproducer is retained from inherited-overload test
isolation. It is not a frozen campaign fixture and is not included in an accepted
compiler capability claim.

JDK 21 (`javac --release 21`, then `java LocalTypeShadow`) prints `true` and exits
zero. The package-private Java class `Parent` becomes Go type `parent`. The legal
Java local `parent` then masks that generated type: subsequent source uses of
`Parent` lower to a Go type reference named `parent` in the variable's scope.

The original larger discovery was saved at
`/private/tmp/java2go-inherited-overload-with-type-shadowing.txt`; its generated
module failed with `parent (local variable) is not a type` at six declarations.
This smaller source preserves the same binding collision without overloads.
The required repair must distinguish type and value namespaces, retain original
Java metadata names, and preserve local-binding behavior. Renaming Java source
variables or changing expectations is not the repair.

The reduced source was verified independently: JDK prints `true`; Go build fails
at `stdjava.ObjectView[*parent](broad, ...)` with
`parent (local variable) is not a type`.
