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

## Futures and thread lifecycle

Fixed and single-thread executors support `submit(Callable<T>)`,
`submit(Runnable)`, `submit(Runnable, result)`, `execute`, `shutdownNow`, orderly `shutdown`,
`isShutdown`, `isTerminated`, and timed `awaitTermination`. Accepted tasks drain
an unbounded queue; submissions after shutdown throw `RejectedExecutionException`.
`Future<T>` supports repeatable `get`, timed `get`, `cancel`, `isDone`, and
`isCancelled`. Task failures become `ExecutionException` with the original Java
throwable as cause. Native Go panics use the existing Java panic normalization.
Callable lambdas, method references, anonymous implementations, and named
implementations retain the worker's logical execution through generated calls.

Timeouts support all seven `TimeUnit` constants, null checks, immediate polling,
and saturated conversion for large values. `Thread.join` returns for unstarted
threads and supports millisecond/nanosecond timeouts; `isAlive` observes lifecycle
state and repeated `start` throws `IllegalThreadStateException`. Uncaught thread
and `execute` task failures are reported to stderr; custom uncaught-exception
handlers are not implemented.

`Thread.currentThread()` and `getName()` use the explicit execution token.
A started thread observes its own identity, direct `run()` retains the caller,
and fixed-pool workers retain their identities across successful tasks.
`CountDownLatch` supports countdown, count queries, blocking and timed waits.

`shutdownNow` returns queued Runnable objects without running or cancelling them.
Tasks queued by `execute` retain their original identity; `submit` returns its
queued Future wrapper, which can still be run after removal. Running workers
receive a cooperative interruption request. Execution-aware latch waits observe
and clear interruption by throwing `InterruptedException`.

Cancellation publishes a terminal state immediately and skips queued work.
`cancel(true)` does not yet request Java interruption. Thread sleep, Future.get,
and monitor waits do not yet observe interruption, and executor termination waits
for running task bodies to return. Scheduling, the full Thread API, explicit
FutureTask construction, and CompletableFuture remain outside this subset.

## Charset and buffer subset

The six StandardCharsets constants, Charset.name/forName, String.getBytes,
String.toCharArray and String constructors from byte[]/Charset and char[] support
ordinary scalar Unicode text. Nonstandard encodings are unavailable. Isolated
surrogate char[] construction explicitly throws UnsupportedOperationException
until the string representation can preserve such Java values; substring also
retains the documented rune-indexing limitation. Malformed UTF-8 decoding is not
yet fully matched to the JDK replacement rules.

Array-backed ByteBuffer wrap/allocate, array identity, remaining, position and
bulk get preserve shared storage and underflow behavior. Direct/read-only
buffers, byte order, marks and primitive views are not implemented.
