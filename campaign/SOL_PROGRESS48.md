# Further dependency and allocation evidence

The minimal native repair has passed the strict application parity shard in the original full22 run. Later regression gates remain active on session38667; inspect its result or poll the same process before any restart.

The canonical String branch now includes UTF16 whitespace/prefix/suffix helpers and typed primitive argument captures. Commit `f377ebb63fe71fbc341dd28b2ddec9b90756a7e4` is pushed to `origin/codex/adversarial-string-checkpoint19`. Nine runtime controls and five generated JVM/race parity controls passed independent review. Separate source-call staging coverage passed both before and after the change; the reproduced failure was on String intrinsic argument capture. New tests were formatted for publication without changing Java or expected string literals.

Duration/Instant factory/getter parity is independently accepted at its exercised scope. Source Entry raw generic storage and native live-entry workflows also have focused green results. Inherited Entry dispatch, broader Set/Map contracts and combined regression checks remain open. The Class factory's seven-source application passed, but independent review found a binder-shadow case in field qualification; its added reproducer is being repaired and revalidated rather than ignored.

Collection/Set defaults are being checked with their complete frozen applications. The StringBuilder prerequisite now supports direct calls, while its full fixture remains red on missing IntConsumer/ObjIntConsumer method-reference contracts. DateFormat has a corrected explicit UTF-8 JVM oracle, preserving CLDR narrow spaces; earlier ASCII stdout observations remain separate evidence.

Allocation profiles are independently accepted for attribution. Each of two modes recorded 10,752 literal-path wrapper objects and 10,609 clone objects (4,601,776 combined bytes) across a process containing21 checked calls. The wrapper count alone demonstrates repeated allocation beyond first use. Four extra clone objects and256 bytes remain unexplained. These whole-process profiles are neither the earlier20-scan measurement window nor CPU/Java performance comparisons. A cache is not yet implemented.

The profile evidence is preserved in `.campaign/checkpoints/checkpoint19/literal-v3-accepted/literal-v3-accepted-attribution.tar.gz` (146 verified members plus manifest), SHA256 `3b1e9a38b0ac18001205fde1edc1e1dcf7addd5a8262d98a089bdd5008376329`. It references the already-preserved v2 source archive, whose hash was reverified.

Checkpoint18 remains the latest production compatibility acceptance. Full Gson is still required and failing; Netty progression remains gated.
