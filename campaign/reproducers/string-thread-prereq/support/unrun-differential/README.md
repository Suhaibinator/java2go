# Frozen String/thread differential runner

This runner is prepared but has **not been executed**. The coordinator owns the
heavy execution slot. The frozen Java input is
`/private/tmp/java2go-string-thread-prereq`, with SHA-256
`206b89997fca211711a90e504d29dc1f314a674cfa6a63cb8f7a53037fc684cf`
over sorted relative path, NUL, file bytes, NUL. Its nine independent JDK21
observations are in `/private/tmp/java2go-string-thread-prereq-validation`.

The runner stages byte-identical source and this pinned Java 21 POM in a private
Maven directory. It checks the frozen16v3 manifest, builds that exact compiler,
transpiles all seven classes in strict Maven mode, builds every generated Go
package with `-race`, builds the app with `-race`, then compares nine fresh Go
processes with the corresponding individual JVM stdout, stderr, and exit code.
First failure and every raw stream are retained outside frozen input and
snapshot directories. It does not execute or change the Java oracle.

Each command starts in a new process session; a timeout kills its process group
and waits for cleanup. Infrastructure failures are recorded separately from
program mismatches. Every Go run has a fresh working directory; any file or
directory it creates is recorded as an unexpected output. The runner pins and
checks its own bytes and the POM via `pin.json`, and verifies frozen input and
compiler snapshot hashes before and after execution. An after-run integrity
change invalidates the result. Command environments explicitly select JDK21,
`GOMAXPROCS=2`, shared `/private/tmp/java2go-campaign-go-cache`, and
`JAVA2GO_ASSERTIONS=false` for default JVM assertion parity.

After the coordinator releases the heavy slot, run:

```sh
python3 /private/tmp/java2go-string-thread-differential/run.py
```

Artifacts will appear under
`/private/tmp/java2go-string-thread-differential/artifacts/` with a terminal
`report.json`. Limits: compiler build 300 seconds, strict transpilation 300,
all-packages race build 300, app race build 300, each Go process 60 seconds.
The runner refuses to replace a prior artifact directory.
