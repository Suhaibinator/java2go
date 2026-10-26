# Sol checkpoint 228: captured local classes inside lambdas

The local AbstractMap application exposed a lost execution context in a computeIfAbsent callback. Its lambda could not resolve a captured local class after the compiler replaced the enclosing method scope with a synthetic SAM scope. Overload selection then emitted a public wrapper that created a fresh execution context.

The repair uses the actual lambda body's ancestry to retain local-class visibility within the exact declaring Java executable. It preserves the existing block binding map and refuses foreign executables or incomplete anchors. No Map or runtime implementation changed.

Normal and race runs pass five parents and seven subcases each, including the unchanged LocalAbstractMap and pollution acceptance fixtures, boundary observations, direct/nested lambda lookup, and negative visibility guards. All-package golangci-lint2.15 passes. Independent source/actual review: `4558d1e5528afbacdedf085fdb76eb0c561dd285b29563d8f4fb7405d5aa414f`.

Published-head CI remains pending. Full Netty/Gson compatibility and the separate Add-contract test migrations remain open.
