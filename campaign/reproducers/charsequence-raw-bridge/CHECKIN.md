# Supplemental inherited CharSequence/raw Iterable bridge probe

Proposed checked-in path: `campaign/reproducers/charsequence-raw-bridge`.

**JDK oracle: VALIDATED. Go/transpiler: UNRUN.** This is a supplemental prerequisite probe, not a replacement for the frozen full Gson application or the original Throwable workflow.

All six Java files, original README, prepared source-hash list, bounded runner, and bounded subprocess helper are copied byte-for-byte from `/private/tmp/charsequence-raw-bridge-probe`. The original README's pre-run status is deliberately preserved; this file records the later JDK validation. `oracle.json`, `commands.txt`, and canonical `expected.seed-{17,41,97}.{stdout,stderr}` are copied from the validated JDK run. `PROVENANCE.json` records all nine process observations and their canonical-stream correspondence. No dependencies, classes, or bulk run logs are vendored.

The Oracle JDK 21 `javac --release 21 -encoding UTF-8` compile exited 0. Each seed ran in three fresh JVM processes; all nine exited 0, produced byte-identical stdout within its seed, and had empty stderr. Prepared Java and README hashes matched before and after. `PACKAGE_SHA256SUMS` covers every package file except itself.

In the first join, an inherited `CharSequence.toString()` replaces a later list slot through a raw `List`. The inherited source iterator then reads that slot; Java's delayed cast throws `ClassCastException`. Seeds 17 and 41 poison slot 1; seed 97 poisons slot 2, allowing one additional conversion. Restoring the slot permits a second join with all elements and ordered conversion events.
