# Supplementary String.join(Iterable) mutation probe

This JDK-only, multi-package probe is independent of the frozen Gson and Throwable workflows. It exercises a one-pass source `Iterable<CharSequence>` and a forwarding source `Iterator` that logs `iterator`, `hasNext`, and `next` calls. A custom `CharSequence` logs `toString`, `length`, `charAt`, and `subSequence`; five cases observe conversion count/order, live `List.set`, an add/remove pair that restores list size but changes `ArrayList` modification count, null-returning conversion, and a throwing conversion with a `finally` marker.

Prepared JDK 21 oracle commands, **not yet executed** (coordinator owns the heavy slot):

```sh
JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
OUT=/private/tmp/string-join-iterable-oracle-20260928
mkdir -p "$OUT/classes"
"$JAVA_HOME/bin/javac" --release 21 -encoding UTF-8 -d "$OUT/classes" src/probe/join/state/*.java src/probe/join/source/*.java src/probe/join/app/Main.java
"$JAVA_HOME/bin/java" -cp "$OUT/classes" probe.join.app.Main 17 > "$OUT/seed-17.stdout" 2> "$OUT/seed-17.stderr"
"$JAVA_HOME/bin/java" -cp "$OUT/classes" probe.join.app.Main 41 > "$OUT/seed-41.stdout" 2> "$OUT/seed-41.stderr"
"$JAVA_HOME/bin/java" -cp "$OUT/classes" probe.join.app.Main 97 > "$OUT/seed-97.stdout" 2> "$OUT/seed-97.stderr"
```

Each seed should run in three fresh JVM processes before its output becomes an oracle.
