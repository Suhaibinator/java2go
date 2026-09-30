# Frozen reflection exception/thread prerequisite

All 14 original input files (`README.md`, three seed inputs, and ten Java
sources) are byte-identical to the private JDK-validated probe. Their framed
combined SHA-256 is `a64df8bc84345ba2c8f4f1f8340cdd33b535e737f1203fd2ef313a07b594a42e`.
The original README is retained unchanged for hash fidelity; its pending-JVM
statement is superseded here.

Explicit JDK21 `javac --release 21 -encoding UTF-8` compiled the full source.
Nine fresh JVM processes, three each for seeds 17, 41, and 97, exited 0 with
empty stderr, no output files, and byte-stable stdout within each seed. The
compact `oracle/oracle.json` records original commands and every exit/stream
hash. One raw stdout/stderr pair per seed is retained under `oracle/`; the
stderr files are empty. No classes or bulk logs are included.

`support/validate_jvm.py` is an exact copy of the original oracle runner; the
**original private runner was executed**, while this packaged copy is **unrun**
and still contains absolute private paths. Its SHA and command provenance are
in `oracle/oracle.json`. Relocate the paths explicitly before re-execution.

**No Go transpilation, Go build, race run, or parity acceptance is claimed.**
