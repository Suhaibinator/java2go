# Sol progress67: erased null and inherited header repairs

The general compiler null-comparison repair routes known reference operands through existing `JavaReferenceEqual`, including erased wildcard returns holding typed nil. It preserves primitive unboxing, single evaluation, object identity and source callbacks. The runtime implementation already handles typed nil; this checkpoint adds regression controls without a runtime behavior replacement.

Inherited generic field typing now resolves explicit superclass arguments in the child declaration header and raw parent bounds in their original parent namespace. It preserves emitted binder identity and prevents same-spelling types in a method body from changing inherited storage typing.

The focused null gate passes all 23 actions. The inherited-header gate passes all six original controls, with source-derived JDK witnesses retained. On the combined 2396-file source (`eb9b56acd5e6b82fee06b252059eb612527a1b84327c73cc5beff9aba5a79f70`), fresh Original11 and Hard3 both pass: 12 and five parent/action records, respectively, with strict translation, all generated package builds, race entry builds and complete observations. The formerly failing bridge null row now equals the JVM `true:1:true`; its subsequent reference observations also execute and match.

Independent audits `47dee5df…` and `903316e5…` verify source/input/tool stability, raw output parity and all 38 application-gate cleanup receipts. The managed publication carries the exact five-path implementation/test delta while preserving newer campaign files from the preceding checkpoint. It is not a whole-source2396 manifest claim or full coherent CI acceptance.

Normalizer project parity independently passes nine complete raced executions; its accepted changes are being composed and require fresh source-bound gates. Legacy main's raw host argument bug remains a distinct repair. Original Codec now has a reproducible Go build failure at `Path.of` canonical String dispatch; its application, dependencies and expectations remain frozen. Gson reflection has verified JVM controls, with compiler/runtime TDD underway. The remaining native CI, String casing and two dedicated performance correctness tracks continue.

Six isolated Go correctness lanes and two JVM-only lanes are authorized after a fresh CPU/memory check. No performance measurement, Netty Go acceptance or full-round promotion is claimed. Checkpoint18 remains the latest full compatibility acceptance. Continue directly from live agent receipts and retain every earlier failure and frozen challenge.
