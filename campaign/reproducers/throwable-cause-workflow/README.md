# Supplemental Throwable cause-construction probe

Independent JDK-only, multi-package probe. It does not replace or modify the frozen Gson application. A cause is constructed on `main`, then `Exception(Throwable)` and `RuntimeException(Throwable)` construct on a named worker. Output includes inherited virtual `toString()`, null-returning `toString()`, throwing `toString()`, exact cause identity, invocation counts, and try-with-resources close/primary/suppressed exception order. The worker is joined before output, so each seed is deterministic.

JDK 21: `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`.

When the coordinator grants the heavy slot, from this directory:

```sh
JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
mkdir -p classes
"$JAVA_HOME/bin/javac" -encoding UTF-8 -d classes src/probe/state/*.java src/probe/cause/*.java src/probe/app/Main.java
"$JAVA_HOME/bin/java" -cp classes probe.app.Main 17 > expected.seed-17.stdout
"$JAVA_HOME/bin/java" -cp classes probe.app.Main 41 > expected.seed-41.stdout
"$JAVA_HOME/bin/java" -cp classes probe.app.Main 97 > expected.seed-97.stdout
```

Validated with `javac --release 21 -encoding UTF-8`; seed 17, 41, and 97 each ran in three fresh JVM processes, exited 0, had empty stderr, and produced byte-identical stdout per seed. Classes, exact commands, status, source hashes before/after, and all stdout/stderr are stored outside this source tree at `/private/tmp/gson-throwable-cause-oracle-20260928`.
