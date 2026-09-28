# Supplemental Throwable cause-construction workflow

Proposed checked-in location: `campaign/reproducers/throwable-cause-workflow`.

This package contains the **original full supplemental Throwable workflow** with exact Java sources and README, plus byte-for-byte JDK 21 expected stdout/stderr for seeds 17, 41, and 97. Each seed was verified across three fresh JVM processes. `oracle.json` records the JDK, commands, exits, hashes, and package-file inventory.

The JDK oracle is green. The full workflow is currently **red in the translator at `String.join(Iterable)`** used by its trace snapshot; retain the workflow unchanged while that prerequisite is repaired. The separately validated smaller named-cause fixture is green and is **not** this package. The unrun independent join/mutation probe is also separate and is **not** included.

Do not replace this workflow with either smaller probe. No class files or bulk run logs are included.
