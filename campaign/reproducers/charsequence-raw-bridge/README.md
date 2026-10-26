# Unvalidated supplemental CharSequence/raw Iterable bridge probe

This small JDK-only probe is a cross-package prerequisite test for nominal `CharSequence` and `Iterable` bridges. `DerivedSequence` inherits `SequenceBase.toString()`; `SequenceSource` inherits `BaseSource.iterator()`. A raw `Iterable` cast reaches `String.join`. The first element's inherited conversion replaces a later list slot with an `Object` through a raw `List`, leaving size unchanged. Seeds 17 and 41 poison slot 1, while seed 97 poisons slot 2 so an additional valid element converts before the delayed `ClassCastException`. After restoring the element, a second join observes all values, conversion counts, and iterator order.

**UNVALIDATED:** Java source has been prepared but no JDK oracle or transpiler run has been executed. This does not replace the frozen full Gson application, the original Throwable workflow, or the validated custom-Iterable probe.

The exact bounded nine-run oracle command, only after the coordinator grants a heavy slot, is:

```sh
/bin/bash /private/tmp/charsequence-raw-bridge-probe/run-oracle.sh
```

The runner uses the JDK below with `javac --release 21`, a 300-second compile bound and a 60-second bound for each of nine fresh JVM processes. On timeout it terminates the subprocess group, verifies the prepared input hashes before and after, compares each seed's three stdout/stderr pairs byte-for-byte, and saves canonical expected streams plus a compact manifest under a fresh `/private/tmp/charsequence-raw-bridge-oracle.*` directory.

Underlying JDK commands:

```sh
probe_jdk=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
probe_out=/private/tmp/charsequence-raw-bridge-oracle
mkdir -p "$probe_out/classes"
"$probe_jdk/bin/javac" --release 21 -encoding UTF-8 -d "$probe_out/classes" src/probe/bridge/state/Trace.java src/probe/bridge/source/*.java src/probe/bridge/domain/*.java src/probe/bridge/app/Main.java
"$probe_jdk/bin/java" -cp "$probe_out/classes" probe.bridge.app.Main 17
"$probe_jdk/bin/java" -cp "$probe_out/classes" probe.bridge.app.Main 41
"$probe_jdk/bin/java" -cp "$probe_out/classes" probe.bridge.app.Main 97
```
