# Supplemental custom-Iterable `String.join` probe

Proposed checked-in path: `campaign/reproducers/string-join-iterable`.

**JDK oracle: VALIDATED. Transpiler: UNRUN.** This is a supplemental prerequisite probe; it does not replace or weaken the frozen full Gson application or the original Throwable workflow.

The Java source, original README, source-hash list, and bounded oracle runner are copied byte-for-byte from `/private/tmp/string-join-iterable-probe`. The README deliberately remains the exact prepared input; its pre-run wording is superseded by this status file. `oracle.json` and `expected.seed-{17,41,97}.{stdout,stderr}` are copied byte-for-byte from the validated JDK run. No classes, temporary build output, or nine-run bulk logs are included.

Oracle JDK: Oracle Java 21 at `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`, compiled with `--release 21 -encoding UTF-8`. Seeds 17, 41, and 97 each ran in three fresh JVM processes; all nine exited 0, each seed's stdout/stderr matched byte-for-byte, and stderr was empty. Prepared input hashes were verified before and after compilation/runs. `oracle.json` records the nine output hashes and compile result. `PACKAGE_SHA256SUMS` records every copied package file except itself.

Observed JDK contracts: custom iterable/iterator calls occur in order; each element's `toString()` runs once. A `List.set` during the first conversion changes the later value read by the same iterator. An add/remove pair restoring list size still makes `ArrayList`'s iterator throw `ConcurrentModificationException`. A null-returning conversion converts the later element before `String.join` throws `NullPointerException`. A throwing conversion preserves the same exception object, skips the tail, and executes `finally`.
