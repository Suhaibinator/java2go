# Adversarial Java → Go campaign runner

`javacampaign` checks complete applications against an independently compiled JVM oracle. It never treats unsupported conversion, missing implementation source, timeout, or a documented known gap as a passing result.

```sh
python3 scripts/campaign/bootstrap.py
JAVA_HOME=/path/to/jdk-21 go run ./cmd/javacampaign \
  -fixture testfiles/campaign/round01/business
JAVA_HOME=/path/to/jdk-21 go test -tags=campaign ./e2e \
  -run '^TestCampaignDifferential$' -count=1
```

The ordinary harness tests run with `go test ./campaign ./cmd/javacampaign`. The `campaign` build tag is explicit selection of the expensive end-to-end suite; its test unconditionally runs every discovered fixture and fails on every remaining discrepancy. It contains no skip or expected-failure mechanism.

## Reproducible dependencies

`dependencies.lock.json` records exact release URLs and SHA256 for Maven 3.9.16, Commons Lang 3.20.0, Commons Codec 1.22.1, Gson 2.14.0, and Gson's required Error Prone annotations 2.48.0. Libraries have matching binary JAR, source JAR and POM artifacts. The bootstrap installs Maven into `.campaign/tools`, archives into `.campaign/cache`, and intact source archives into `.campaign/sources`. Upstream license/notice files remain in archives and extracted trees.

`maven.lock.json` pins every resolved POM and JAR used by the Maven compile lifecycle, including compiler plugin 3.16.0 and resources plugin 3.5.0. Bootstrap downloads those files into `.campaign/m2` and rejects checksum mismatches. Every real fixture Maven build runs **offline** with that project-local repository. It does not write `~/.m2` or invoke Maven against the original fixture directory.

The runner rereads source implementations directly from each verified archive. An edited extracted `.campaign/sources` tree cannot change the translation. A declared selection is a **frozen selected whole-class implementation closure**, not a claim that the whole dependency library translates. Every selected source file is copied unchanged and compiled in full, including unused methods and optional paths. `"**"` explicitly selects every Java source in an archive, including module descriptors; unresolved or unsupported paths fail and are not pruned. Binary dependencies cannot make the source-closure compilation succeed because that compilation has a JDK-only classpath.

## Fixture contract

Each fixture directory contains `fixture.json`, the original POM, sources/resources, and exact `expected.seed-17.stdout`, `expected.seed-41.stdout`, `expected.seed-97.stdout` files created from the stabilized Java oracle. A nonempty stderr requires the corresponding `.stderr` files. Absent stderr snapshots mean exactly empty stderr.

```json
{
  "name": "round01-example",
  "main_class": "campaign.example.Main",
  "source_roots": ["src/main/java"],
  "pom": "pom.xml",
  "dependencies": ["commons-codec"],
  "dependency_sources": {
    "commons-codec": [
      "org/apache/commons/codec/BinaryDecoder.java",
      "org/apache/commons/codec/BinaryEncoder.java",
      "org/apache/commons/codec/CharEncoding.java",
      "org/apache/commons/codec/Decoder.java",
      "org/apache/commons/codec/DecoderException.java",
      "org/apache/commons/codec/Encoder.java",
      "org/apache/commons/codec/EncoderException.java",
      "org/apache/commons/codec/binary/Hex.java"
    ]
  },
  "resources": [{"source": "src/main/resources", "target": "."}],
  "args": ["{seed}"],
  "seeds": [17, 41, 97],
  "repeats": 3,
  "output_files": ["out/report.txt"]
}
```

All paths are fixture-relative, except package-relative dependency source paths. IDs match dependency lock IDs. Every declared dependency must have matching exact direct coordinates in the fixture POM; make any needed transitive source dependencies explicit. The POM must pin compiler/resources plugins supported by the Maven lock. Seeds and repeats are mandatory and cannot be reduced. Unknown manifest fields, empty source roots, duplicate sources, undeclared dependencies, and escaping paths fail.

Resources may be files or directories. They are copied to each fresh process working directory and to the Java classpath. Write output files relative to that directory. `{seed}` substitution applies only to arguments.

Fixtures producing files must also contain a frozen `oracle.json` with the Java observations:

```json
{"runs":[{"seed":17,"repeat":1,"exit_code":0,"stdout":"...\n","stderr":"","files":{"out/report.txt":"<SHA256>"}}]}
```

Include all three seeds; repeated entries must agree. Nonzero expected exit codes can also be declared there. Its streams must exactly match the text snapshots. The runner never rewrites these files. Output-file names and contents (SHA256) must match; all additional created or modified files are compared too, and input resource deletion is observable. Raw output files remain available in run artifacts.

## Gates and retained evidence

Each run gets a unique `.campaign/runs/<timestamp>-<suffix>` directory. Build stages default to five minutes each; program executions default to 60 seconds each. Override deliberately with `-build-timeout` and `-run-timeout`. `-jdk` takes precedence over `JAVA_HOME`, followed by discovery of JDK 21. The selected Java executable must report specification version 21. Java and javac are invoked through that explicit installation.

The runner performs these mandatory gates:

1. Verify dependency locks and freeze fixture, source, resource, original POM, and oracle hashes. Capture Git revision plus content hashes of compiler/runtime/harness sources, including dirty files.
2. Build an isolated copy of the original project with the original POM using offline Maven `compile`.
3. Compile every declared app and dependency source with JDK-only `javac`, checking the complete declared compile-time source closure.
4. Compile app sources against the exact published dependency binaries and run each seed three times. Check the frozen snapshots and repeat determinism.
5. Run the source-built Java graph with the same nine seed/repeat pairs and require parity with the published-binary JVM oracle.
6. Build and execute a separate transpiler process in strict mode. The transpiler consumes an explicit generated source-only POM to ingest the frozen sources. **This boundary does not claim translation of the original Maven build/plugin semantics.** Original POM validation is the separate Maven gate.
7. Compile **all** generated Go packages, then build `./cmd/app`. Execute each seed three times with a separate working directory and compare exact exit code, stdout bytes, stderr bytes, and output files.
8. Recheck fixture hashes, dependency locks, and compiler/runtime/harness content hashes. Changed inputs invalidate the run even if the observed outputs matched.

`report.json` provides structured stages, command lines, toolchain, selected dependency mode, fingerprints, observations, and failure classification. `prior_failure` preserves the original failure when a later integrity check invalidates it. Per-stage logs and raw per-execution `.stdout`/`.stderr` files retain exact bytes, including non-UTF-8 output; JSON text is a readable summary. Generated code, JVM classes, dependency sources, Maven output, and process output files remain in the run directory for independent inspection. No cleanup removes failed artifacts.

Use `-repository` for another checkout, `-artifacts` for a different artifact parent, or `-transpiler` for an already built executable; the report hashes the transpiler binary. To rerun, invoke the same fixture command after restoring its locked source/oracle files. A successful report always covers all nine Java/Go pairs.

## Java assertions

Generated programs support Java `assert` with a process-start setting:
`JAVA2GO_ASSERTIONS=true` enables all translated source assertions. The default,
and `JAVA2GO_ASSERTIONS=false`, disables them. The setting is read once during
runtime initialization. Conditions are not evaluated when disabled; detail
expressions and their text conversion run only for a failed enabled assertion.
Failure throws `AssertionError`, preserving a Throwable detail as its cause.
This is a global mode, not the JVM's per-package, per-class, system-class, or
ClassLoader assertion configuration syntax. Those scoped modes are unimplemented.
The campaign runner explicitly selects `false` for its default-disabled JVM
oracle, regardless of the invoking shell's `JAVA2GO_ASSERTIONS` value. Dedicated
JVM parity tests also exercise enabled assertions with `java -ea`.

Generated Go must be regenerated whenever its compiler/runtime revision changes.
The campaign always rebuilds against the matching local runtime. In particular,
`Supplier<T>` now uses an execution-aware `stdjava.Supplier[T]` interface instead
of a plain Go function, so callbacks created on one Java thread execute with
the calling thread's identity and ThreadLocal values. Previously generated Go
using the old Supplier ABI must be regenerated before linking this runtime.

Failed Maven project conversion retains its private `.java2go-project-*` staging
folder beside the requested output, inside the campaign run directory. Error
messages name that folder, preserving source and generated Go paths for diagnosis.
Panic unwinding also retains staging and reports its path without replacing the
panic. Successful publication cleans staging normally. Retained diagnostic output
is never treated as a successful conversion or parity result.

## Current character and reflection boundaries

Focused JVM regressions cover source and anonymous Reader/Writer subclasses,
UTF16 `char[]` dispatch, Appendable overrides, Writer's default locking and
reusable buffer, null-preserving reference casts, and StringBuilder UTF16 edits
and reversal. These do not establish every JDK character-stream contract:
builtin reader/writer adapters, Reader skip/mark/default lock behavior,
StringBuffer synchronization and distinct class identity still need work.
Isolated surrogate units can be stored in a builder, but conversion to the
current String representation remains explicitly unsupported.

Source implementations of ParameterizedType, GenericArrayType, WildcardType,
TypeVariable and related protocols retain nominal checks, virtual dispatch,
array identity and exceptions. Actual generic Class/Field metadata is a separate
unimplemented prerequisite; see [the metadata design](design/generic-reflection.md)
and [the erased factory design](design/nested-generic-factory.md). Passing these
focused tests does not imply that Gson or arbitrary reflection is supported.
The frozen Gson application remains a mandatory failing campaign gate until its
complete translated implementation builds and matches all JVM observations.


`Function<T,R>` also uses an execution-aware object interface. Regenerate prior
Go output before linking the checkpoint15 runtime. Focused JVM tests cover
nullable values, receiver/argument evaluation order, named/inherited SAM
implementations, object identity and invoking-thread callbacks. These cases do
not establish support for every raw/wildcard conversion or Function default API.
