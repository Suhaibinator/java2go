# Sol TDD progress checkpoint

The native full regression run passed build, lint and all 1,930 unit records, then failed one of 54 application parity cases. Ten later gates were not run. The failure was reproduced and traced to static import scanning pairing one Java file's AST with another file's source bytes.

The general fix reads the import AST from the declaring file. The original application, a diagnostic reproducer and explicit/wildcard import-identity controls pass. Independent review accepted this focused repair. The code and regression test are committed and pushed as `e92116fb294123230113e728217b4b5875bdb9fc` on `origin/codex/adversarial-pause-checkpoint19`. A fresh full22 run is prepared on the minimal repaired source; it is not yet accepted.

Additional TDD results:

- String whitespace helpers passed nine runtime controls and three generated Java/Go parity cases, with independent review.
- Source Map.Entry reference behavior passed its focused JVM/strict/race-build/Go comparison. Native live entries remain red at compilation and have a separate Astra owner for identity and lifetime semantics.
- Five independent JVM prerequisite oracles passed for Entry, Class, AbstractCollection, AbstractSet and corrected AbstractMap. An invalid initial Map driver remains recorded separately.
- Scalar Class unit repairs passed, but the full factory application remains red on source generic constructor representation. The next focused constructor proof has a recorded red and prepared repair.
- Abstract collection defaults have valid JVM oracles and reproducible generated-Go failures. Iterator removal, nominal interfaces and inherited JDK methods remain prerequisites.
- Duration/Instant implementation now compiles past missing types; the standalone test harness still needs an additive Go entrypoint. This is not accepted parity.
- Prefix/suffix testing exposed a general Java int argument-capture issue with Integer.MIN_VALUE/MAX_VALUE. The original frozen test remains active.

Normal optimizing compiler diagnostics independently confirm allocations for the intern lookup's JavaString wrapper and cloned UTF16 buffer. The 64 generated UTF16 composites do not escape. Allocation profiles are a separate exclusive gate; there is no optimized result or claim of superiority over Java.

Selected completed evidence is preserved outside tracked source in `.campaign/checkpoints/checkpoint19/sol-progress47/evidence.tar.gz`: 176 files, archive SHA256 `bcbb0c0d2b7d690989907a8095c23ab37174e6d180fe8ea021240a7fc2488ae6`, manifest SHA256 `547239089d360de99bfc492704b157a2cb0124568006ae2bf2f3c21e0fc49c97`.

Full Gson remains the required failing application. Checkpoint18 remains the last production compatibility acceptance. The campaign continues with twelve Sol agents, Astra on the hardest problems, three isolated correctness lanes, and exclusive performance measurements.
