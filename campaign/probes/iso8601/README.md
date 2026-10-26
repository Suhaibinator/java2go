# Upstream ISO8601Utils integration probe

Run from a bootstrapped campaign checkout with JDK21:

```sh
JAVA_HOME=/path/to/jdk21 python3 campaign/probes/iso8601/run.py
```

The script requires the project-local Gson sources archive whose checksum is in
`campaign/dependencies.lock.json`. It verifies the archive and copies the exact
whole `ISO8601Utils.java` member into a temporary Maven project without modifying
its package, imports, methods, or parsing logic. The checked-in driver controls the
default timezone and exercises valid parses, fractional seconds, offsets, strict
calendar failures, parse-position/error-offset behavior, and all formatting overloads.

The JDK21 oracle and strict generated Go executable must produce byte-identical
stdout, stderr, and exit status. Build commands have 300-second bounds and executions
have 60-second bounds. The recorded java/javac versions must both be JDK21.
All generated packages build; the executable is race-instrumented. Commands,
archive/member/driver hashes, implementation revision and source fingerprints before
and after (which must remain identical), and raw stdout/stderr are retained below
`.campaign/probes/iso8601-*`. This is a focused integration probe, not acceptance of
the full frozen Gson challenge, and never changes that challenge or its oracles.
