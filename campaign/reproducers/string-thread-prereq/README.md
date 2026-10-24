# String interning and worker identity prerequisite

This is a private, source-only supplement to the frozen String ABI and
ConcurrentMap probes. It is not a campaign round. Its seven classes are
separated into app, flow, literal, and state packages. The two
single-thread executors provide stable worker ownership; latches enforce the
ordered `A:intern > B:compare > B:intern > A:complete` trace. Future joins
order the subsequent persistent and reset callbacks. Output observes the
actual trace without sorting or using sleeps.

The two workers create equal dynamic strings through different allocations,
compare them by value and reference, and share the interned object. A worker
also interns a fixed char-array value before `LateLiteral.value()` executes.
Unpaired UTF-16 high and low surrogates are compared as code units and printed
as integers. ThreadLocal state persists on each dedicated worker, remains
isolated from the caller and other worker, then changes identity after remove.

Oracle validation is intentionally pending the coordinator's execution slot.
Use JDK 21 `javac --release 21` and fresh JVM processes for three repetitions
of each seed 17, 41, and 97. Compare exact stdout, stderr, and exit status and
check source hashes before and after. No expected-output file is frozen yet.
