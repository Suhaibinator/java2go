# Round 01 frozen input and oracle

JDK: `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home` (Java 21.0.6). Dependency: real Apache Commons Lang 3.20.0 JAR for JVM runs; the exact upstream `Mutable.java`, `MutableInt.java`, and `MutableLong.java` source files for translation. The selected three files compile with JDK-only classpath along with all 13 application Java files.

The `challenge` SHA-256 below hashes every application `.java` file in sorted path order, then `pom.xml`, then `fixture.json`. Each path and file body are separated by a NUL byte. `dependency` uses the three selected upstream source files in the order listed above with the same encoding. `oracle` uses `expected.seed-17.stdout`, `expected.seed-41.stdout`, and `expected.seed-97.stdout` in that order.

| Input | SHA-256 |
| --- | --- |
| challenge (15 files) | `897fac79c1ee7eee8b58571e3886949c8c4584a2ad4ace8910fb757e79ad485f` |
| dependency (3 files) | `82d02fe4b6130f73ff3f30cd9a7c62771df6cfa293a99ae34654219ca641e65c` |
| oracle (3 files) | `8efcc0529af6d03e79dd90da5e41841a36c308452f3fe9d6869038eebccbf87b` |

Each seed was run in three fresh JVM processes with exactly equal stdout and empty stderr. The source-compiled dependency and published JAR produced byte-identical outputs.

| Seed | Each of three stdout SHA-256 digests |
| --- | --- |
| 17 | `4d8b58ee587e20fda14f86c686d063369f390d972fbaff65c6c36f82e3ef5810` |
| 41 | `657b011bf03bc4d4ebaee82f556cc6575614d73daa795ed4ed878555bcb0bb33` |
| 97 | `976839f47202299610c60f6eb1b7f3ac4022c03b1a9fe783e77d740bb22fbacf` |

Initial manual conversion used a source-only copy of the fixture POM in `/tmp/java2go-business-app-source` and a source-only POM around the exact three Apache sources in `/tmp/java2go-business-lang-source`:

```sh
GOCACHE=/tmp/java2go-business-gocache go run ./cmd/java2go \
  -maven /tmp/java2go-business-app-source \
  -dependency-source org.apache.commons:commons-lang3=/tmp/java2go-business-lang-source \
  -main-class campaign.business.app.BusinessApplication \
  -runtime /Users/suhaib/.codex/worktrees/adversarial-java/java2go \
  -module campaign.test/business \
  -output /tmp/java2go-business-go-2
```

Conversion completed. After the first SCC inherited-field fix, `go run -mod=mod ./cmd/app 17` in the generated module failed at Go compilation with `j_org/j_apache/j_commons/j_lang3/j_mutable/Mutable.go:6:2: undefined: Supplier`. The upstream declaration is `public interface Mutable<T> extends java.util.function.Supplier<T>`; generated Go embeds unqualified `Supplier[T]`. The application and dependency inputs stayed frozen.

The campaign harness independently reproduced this at `.campaign/runs/20260928T035418Z-3121391865/report.json`: original-POM offline Maven build, selected-source JDK compile, all nine JVM oracle runs, and transpilation passed; `go-build-all` failed on the same `Supplier` line. Its implementation hash was equal before and after the run.
