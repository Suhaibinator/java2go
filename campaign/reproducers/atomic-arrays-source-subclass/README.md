# Atomic array source subclasses: preserved build failure

Status: RED/pending. This portable source is outside selected CI and the bounded atomic-array service acceptance. Both JDK classes are public and non-final. No final-class fiction or subclass support was added.

Original JDK21 compiles the unchanged Java source and prints `17:23:true:true` with a trailing newline, empty stderr and exit0 on three fresh repeats. `oracle.json` pins its source and observations. Strict Java-to-Go translation succeeds; every generated package race build fails. The exact original complete test log and all emitted build diagnostics remain in ignored campaign evidence. `oracle.json` and `probe.json` reference their absolute durable paths and SHA-256 hashes: unresolved native superclass embeddings, superclass execution constructors, and inherited get/set. `probe.json` records pending scope and source/generated/failure hashes.

Reproduce using JDK21 and separate temporary directories:

```sh
JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
atomic_probe=$(mktemp -d)
"$JAVA_HOME/bin/javac" --release 21 -encoding UTF-8 -d "$atomic_probe/classes" campaign/reproducers/atomic-arrays-source-subclass/src/main/java/probe/Main.java
"$JAVA_HOME/bin/java" -cp "$atomic_probe/classes" probe.Main
GOMAXPROCS=2 GOFLAGS=-p=1 go run ./cmd/java2go -strict -maven campaign/reproducers/atomic-arrays-source-subclass -main-class probe.Main -runtime "$PWD" -output "$atomic_probe/generated"
(cd "$atomic_probe/generated" && GOMAXPROCS=2 go build -race -p 1 -mod=mod ./...)
```

Keep compiler/generated build deadline300s, native test deadline55s under a65s supervisor, and each execution deadline60s. No oracle rewrite, generated-code patch, JVM delegation or source-name specialization is permitted. Repair requires a separately owned native-superclass compiler scope; it must preserve source-subclass identity and real constructor/inherited dispatch semantics.

The unchanged original probe wrapper is preserved as `probe.go.txt` for a separately authorized compiler probe; it is not a selected CI test. Publication v2 changes diagnostic provenance metadata and documentation only. Java, POM, expected stdout/stderr, and all three JDK observations are preserved byte for byte. Verbose failure logs are excluded from tracked leaves.

Durable diagnostic evidence:

- full_failure: `/Users/suhaib/code/java2go/.campaign/resume-20261002/sol_gson_type_frontiers/evidence/source-subclass-frontier.log`; SHA-256 `aec98fe8c557c0a35eda5e128f575b999c16bc5f242bf1e757b61762df8b7264`.
- generated_build_stderr: `/Users/suhaib/code/java2go/.campaign/resume-20261002/sol_gson_type_frontiers/evidence/source-subclass-build.stderr`; SHA-256 `2f083d59882097a1a09f9147905372426c3b866a40dd0ca285c068d1f24e7404`.
- generated_go: `/Users/suhaib/code/java2go/.campaign/resume-20261002/sol_gson_type_frontiers/evidence/source-subclass-frontier/TestCampaignAtomicArraysSourceSubclassFrontierJVM/generated/j_probe/Main.go`; SHA-256 `881f04276e5504e44ce7068cd26a542e530afb9ed5ae174feb9ee9bc28691386`.
