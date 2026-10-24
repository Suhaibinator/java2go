# Commons Codec resource ingestion prerequisite

This fixture reads the unchanged Commons Codec 1.22.1 `Resources.java`
implementation and the locked published binary's `dmrules.txt` resource. It checks
absolute and package-relative resource lookup, exact byte count and checksums,
and a missing resource. It covers this resource ingestion path; it does not
establish whole Commons Codec compatibility.

The dependency lock already pins `commons-codec:commons-codec:1.22.1`:

| Artifact | SHA256 |
| --- | --- |
| Published binary JAR | `78a5d732fbd715e2d10bd7150d2f8030bae57267f8aacc5c88f642cb6c2e5d3f` |
| Source JAR | `96444ee2fa5a41dc4d3b7b95f46b67495795d458e2050fc08bb88ab7f87ac021` |
| Original `org/apache/commons/codec/Resources.java` | `f99b811371ba66d9ba2221006901cca90b4e66bc172b48fda5f0eab9700c0c74` |
| Binary `org/apache/commons/codec/language/dmrules.txt` | `c078af51083ddb2d3b9c797e1ee53054b989039d798be2c917791f9e609a630f` |

The rule payload contains 3,546 bytes. Source and binary JARs contain identical
resource bytes; the campaign's runtime payload authority remains the binary JAR.
The manifest selects the entire original `Resources.java` class, with no handwritten
library replacement, and exactly this binary resource path. The resource is frozen
inside its dependency module, hashed, and supplied identically to the source JVM,
published-binary JVM and generated Go application.

The six stream snapshots and `oracle.json` were derived from nine repeated actual
source/binary JVM pairs at seeds 17, 41 and 97. They remain unchanged. The original
resource-omitting harness failed at `Resources.java:41` before transpilation. A
separate regression showed that fixture file target aliases could evade literal
resource collision checks; canonical destination checks reject those aliases.
The final producer passed the original resource tests, nine actual
source/binary/generated comparisons, and three separate runs with race instrumentation.
The publication composition must receive its own fresh verification after rebasing.

After the normal campaign bootstrap, run from the repository root with JDK 21:

```sh
GOMAXPROCS=2 GOFLAGS=-p=1 go test ./campaign -run '^TestDependencyResource' -count=1 -timeout=45s
GOMAXPROCS=2 GOFLAGS=-p=1 go run ./cmd/javacampaign -repository . -fixture campaign/reproducers/codec-resource-ingestion -artifacts .campaign/artifacts -jdk "$JAVA_HOME" -build-timeout 120s -run-timeout 10s -race -stress-runs 3
```

The campaign preserves original Maven compilation, source closure compilation,
published binary observations, actual source translation, generated package builds,
and stdout/stderr/file comparisons. It reports unsupported constructs, resource
selection errors, collisions, build failures and parity differences as failures.
It does not regenerate the frozen oracle or expected streams.
