# stdjava runtime

`stdjava` supplies Java behavior used by generated Go: object and array identity,
checked reference operations, boxed primitives, numeric operations, exceptions,
strings, collections, streams, Optionals, I/O, and concurrency adapters. It is a
runtime for the supported transpiler subset, not a complete JDK implementation.

## Boxed primitives

`Boolean`, `Byte`, `Short`, `Character`, `Integer`, `Long`, `Float`, and `Double`
are distinct immutable pointer objects with private payloads. A nil pointer is
Java null. `BoxInteger(int32)` and the corresponding typed helpers perform
boxing; `UnboxInteger(*Integer)` and its equivalents throw
`NullPointerException` for null. `NewInteger` and the other constructors always
allocate fresh objects.

Boxing caches both Boolean values, every Byte value, Character 0–127, and
Short/Integer/Long −128–127. Float/Double and values outside those ranges allocate
new objects. Character stores a 16-bit Java value while its accessor uses the
generated primitive `rune` boundary. Wrapper methods implement numeric accessors,
value equality, Java hashes, ordering, and string conversion. Floating equality
and hashes canonicalize NaNs and distinguish positive and negative zero.

`JavaNumber` is the interface for Java `Number` references and generic bounds;
its six numeric accessors retain each source wrapper's conversion semantics.
`JavaPrimitiveNumber` is the separate scalar constraint for runtime arithmetic.
Primitive streams and primitive Optionals keep scalar payloads. Boxing streams
and generic Optional values use wrapper pointers.

Collections normalize boxed keys by Java wrapper kind and value while preserving
the first inserted key object. This supports equal, separately allocated wrapper
keys and heterogeneous Object queries. It does not implement arbitrary
user-defined `equals`/`hashCode` for every collection operation. Existing container
null policies and documented eager/sequential stream behavior still apply.

## Arrays and generated API migration

Primitive arrays use `*PrimitiveArray[T]`; reference arrays use
`*ReferenceArray` with runtime component descriptors. Generated Java varargs
parameters use these same array references. Fixed-array calls preserve the
original array or null; expanded calls allocate new arrays. Reference arrays
retain Java's covariant store checks.

Regenerate all output when adopting this runtime revision and update handwritten
Go callers. For example, a Java `List<Integer>` now holds `*Integer` values, and
an `int...` method accepts `*PrimitiveArray[int32]`. Old scalar wrapper values
and generated Go variadic slices are not supported as a legacy output mode.
General reflection, serialization, full Unicode String fidelity, and additional
JDK APIs remain outside this migration.
