# Differential-fuzzer corpus

Each subdirectory holds minimized Java programs that diverged between the JDK and
java2go's transpiled Go, grouped by failure category:

- `transpile_crash/` — java2go panicked or errored converting the program.
- `go_compile_error/` — the transpiled Go did not compile.
- `output_mismatch/` — both sides ran but printed different stdout.

For each `<name>.java` there are two siblings:

- `<name>.expected` — the JDK's stdout (the oracle), or the failure detail.
- `<name>.actual` — the Go side's stdout, or the Go compiler / transpiler error.

The file name is `seed<N>_<hash>`: seed `N` regenerates the *original* (un-shrunk)
program via `go run ./cmd/javafuzz -seed N -only`; the `.java` here is the
*shrunk* minimal repro. The hash is of the shrunk source, so re-finding the same
minimal repro does not create duplicates.

`go test ./fuzz/` replays every entry: programs whose underlying bug is still
open (signature listed in `fuzz/known.go`) are skipped with a reason, mirroring
the e2e suite; programs whose bug has been fixed must now match the JDK, so any
reintroduced divergence fails the test.
