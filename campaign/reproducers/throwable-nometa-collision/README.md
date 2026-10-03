# No-metadata ordinary-object/Throwable-method collision probe

**PREPARED, UNRUN.** This is a small additive control for the existing structural Throwable review. It does not edit or replace the original or prefix probes, nor the frozen full Gson or Throwable workflows. JVM expected streams remain unset until an authorized JDK run; no output is guessed.

The ordinary source class `probe.nometa.Lookalike` has public `throwableTypeName()`, `message()`, and `error()` methods with counters. It does not extend `Throwable` or override `toString()`. `hashCode()` returns 42 and counts calls. The app derives expected default-object text from the literal source-qualified name `probe.nometa.Lookalike` and `Integer.toHexString(value.hashCode())`; it never requests class metadata or uses reflection. It exercises `String.valueOf(value)`, inherited direct `value.toString()`, repeated `String.valueOf(Object alias)`, and `alias.toString()`, printing values, equality, and counters after each conversion. Seeds 17 and 41 perform three alias reads; seed 97 performs two.

Planned JDK 21 command, **only after a heavy slot is granted**:

```sh
probe_jdk=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
probe_out=/private/tmp/throwable-nometa-collision-oracle
mkdir -p "$probe_out/classes"
"$probe_jdk/bin/javac" --release 21 -encoding UTF-8 -d "$probe_out/classes" src/main/java/probe/nometa/Lookalike.java src/main/java/probe/app/Main.java
"$probe_jdk/bin/java" -cp "$probe_out/classes" probe.app.Main 17
"$probe_jdk/bin/java" -cp "$probe_out/classes" probe.app.Main 41
"$probe_jdk/bin/java" -cp "$probe_out/classes" probe.app.Main 97
```

Each seed should run in three fresh JVM processes before any oracle is frozen. A later Go differential must use these unchanged sources and real JDK observations.
