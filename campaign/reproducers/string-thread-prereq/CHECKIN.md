# Frozen String/thread prerequisite package

The original input files (`README.md`, `inputs/`, and `src/main/java/`) are
byte-identical to the private JDK-validated probe. Their combined input SHA-256
is `206b89997fca211711a90e504d29dc1f314a674cfa6a63cb8f7a53037fc684cf`
using the framing specified in `oracle/oracle.json`. The original README is
retained unchanged for hash fidelity; its statement that validation is pending
was true at preparation time and is superseded by this note.

JDK 21 `javac --release 21 -encoding UTF-8` succeeded. Nine fresh JVM runs,
three each for seeds 17, 41, and 97, exited 0 with empty stderr. Output was
byte-identical within each seed. `oracle/oracle.json` records each run and
portable reproduction commands; `oracle/expected.seed-*.stdout` contains one
raw representative per seed (stderr files are empty). No class files or bulk
logs are included.

`support/unrun-differential/` preserves a separately prepared frozen16v3
differential runner and pinned Maven POM. That runner has **not been executed**.
It still names its original `/private/tmp` source, oracle, and snapshot paths;
port those paths explicitly before using a checked-in copy. Do not interpret
its presence as Go parity or any transpiler result.
