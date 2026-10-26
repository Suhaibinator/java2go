# Reflection exception and executor callback prerequisite

This private multi-package source is a supplemental prerequisite, not a new
active campaign application. It is **JVM unvalidated** until a coordinator
assigns an execution slot. No expected-output file exists yet.

Two dedicated single-thread executors run subclasses that inherit the generic
`ReflectiveFactory<T>.call()` owner. Its callback reflectively resolves and
invokes a `(String, Throwable)` constructor. The success path observes the
constructed exception's class, message, cause class and message, and exact
cause reference. The second path deliberately resolves a missing constructor,
observing the reflection exception's class, nonempty message, and null cause.
A try-with-resources lease closes on both paths. ThreadLocal callback contexts
are isolated and removed; follow-up callbacks on the same workers verify fresh
context identity. Latches define the printed event order. No sleeps, GC timing,
network, or sorted race output are used.

The separate private runner at
`/private/tmp/java2go-reflection-exception-thread-runner/validate_jvm.py` will
use explicit JDK21 with `javac --release 21 -encoding UTF-8`, a 300-second
compile bound, and three fresh JVM processes for
each seed 17, 41, 97 with 60-second execution bounds. It records exact
stdout/stderr/exit, input hash before and after, and refuses to freeze an oracle
unless all nine executions are byte-stable per seed. **Do not run until the
coordinator releases the heavy slot.**
