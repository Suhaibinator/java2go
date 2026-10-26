# Supplemental generic forwarding Iterator cast-boundary probe

**SOURCE ONLY — JDK oracle and Go differential unrun.** This multi-package probe is a prerequisite for the existing full Gson, Throwable, and raw-bridge workflows. It does not replace or edit them.

`RawCursor.next()` advances the cursor and records `backing.next`. `Forwarder<T>.next()` has an unbounded type parameter, stores `backing.next()` into a local `T`, then records `forwarder.after-backing.next` before returning. A raw `Iterator<?>` is converted to `Forwarder<CharSequence>` only at the iterator interface boundary. The first raw `Token` is read into `Object`; a seed-dependent number of raw tokens are read with ignored results; both must succeed without a cast. The next raw token is assigned to `CharSequence`, so its `ClassCastException` must occur only after the cursor and forwarder effects. A later null assigned to `CharSequence` succeeds, followed by a valid string tail. All events are visible in one deterministic output line. `Trace` deliberately avoids `String.join`.

Seeds 17 and 41 use three ignored reads; seed 97 uses two. Each seed will require three fresh JDK 21 processes and exact byte agreement before an expected-output file can be frozen. No expected output is fabricated.

Planned JDK: `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`; compile with `javac --release 21 -encoding UTF-8`, then run `probe.forward.app.Main` with arguments `17`, `41`, and `97`. **Do not execute until the coordinator grants the heavy slot.**
