# Generic binding supplemental prerequisite

Proposed repository location: `campaign/reproducers/generic-binding-prereq`.
The original Java sources, README, POM, seed inputs, and dependency contract
are preserved byte-for-byte. This is a supplemental prerequisite, not a new
accepted full campaign application. The existing frozen applications and their
expected outputs remain unchanged.

The real Apache Commons Lang 3.20.0 binary was used for the JDK21 oracle. Its
pinned SHA-256 and the exact three-file source-only implementation closure are
recorded in `oracle/manifest.json`; dependency binaries and sources are not
vendored. All nine fresh JVM runs (seeds 17, 41, 97, three repetitions each)
exited 0 and matched exactly within each seed. Canonical raw streams are in
`oracle/expected.seed-*.stdout` and `.stderr`.

`oracle/runner.provenance.py` is the exact private runner used for validation.
`oracle/run_jdk.py` is a relocated, source-hash-preserving runner prepared for
check-in; it has not itself been executed. Supply a verified Commons Lang jar
using `--jar`, set `JAVA_HOME` to JDK21, and choose a fresh `--out` outside this
package when rechecking. Transpiler status: UNRUN. No Go parity is claimed.
