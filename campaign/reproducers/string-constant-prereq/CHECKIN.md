# String constant and class-initialization prerequisite

Proposed repository location: `campaign/reproducers/string-constant-prereq`.
This is a small, independently validated Java 21 prerequisite, not an
accepted campaign application. It preserves the original frozen README, Java sources, POM, and three seed
input files byte-for-byte. The oracle directory contains
one exact stdout and stderr file per seed; each represents three matching fresh
JVM runs. `oracle/manifest.json` records hashes and the original commands.

Current Go status: the original probe is red at the overflow/constant-folding
boundary. The compiler's typed-evaluator candidate is partial and is not an
accepted parity result. No Go outputs or generated classes are included here.
The unchanged full campaign applications remain the acceptance target.

For a portable JVM recheck, set `JDK21` to a Java 21 installation and choose an
output directory outside this package. Compile all five Java files under
`src/main/java` using `"$JDK21/bin/javac" --release 21 -encoding UTF-8 -d
"$OUT/classes" ...`, then run `"$JDK21/bin/java" -cp "$OUT/classes"
prereq.constant.app.Main 17` (also 41 and 97) in fresh processes. Compare
exact stdout, stderr, and exit code with the oracle files and manifest.
