# Generic factory/cache supplemental prerequisite

Proposed repository location: `campaign/reproducers/factory-cache-prereq`.
The original Java sources, README, POM, dependency contract, and seed inputs
are byte-identical to the independently validated fixture. This remains a
supplemental prerequisite for the nongeneric TypeAdapterFactory generic-method
ABI blocker. It does not replace or weaken frozen full Gson acceptance.

JDK21 oracle status: validated. All nine fresh JVM runs (seeds 17, 41, 97,
three repeats each) exited 0 with exact repeat-stable stdout and empty stderr.
`oracle/expected.seed-*.stdout/.stderr` contain the raw canonical streams;
`oracle/manifest.json` records all nine observations and source hashes.

The real pinned Commons Lang 3.20.0 binary jar was used for JVM compilation
and execution. `oracle/dependency-lock.json` preserves its exact binary and
sources-jar lock entries. The two-file Mutable.java/MutableInt.java whole-class
source closure is recorded with SHA-256 in the manifest. No dependency jar or
source implementation is vendored.

`oracle/runner.provenance.py` is the exact runner used for JDK validation.
`oracle/run_jdk.py` is a relocatable copy prepared for later check-in, not yet
executed. Supply a verified Commons Lang jar through `--jar`, set `JAVA_HOME`
to JDK21, and choose a fresh output directory outside this package. Transpiler
status: UNRUN; no Go parity is claimed.
