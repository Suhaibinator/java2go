# Number implementation checkpoint and continuing adversarial repairs

The Number-family implementation and its complete eight-file Java regression are committed and pushed as fa5ca2e151025e5bff9ee2012eb026e8d58952cb on origin/codex/adversarial-number-checkpoint19. The portable test runs all nine original JVM observations, strictly transpiles, race-builds all generated packages, and compares nine fresh Go processes. The old implementation reproduces the missing ancestor bridge panic; the candidate passes. Thirteen related controls also passed. All1903 non-ledger source files match the tested source manifest. The existing String and native WIP branches remain preserved.

This is a WIP checkpoint. Independent review found that explicit Number-bound casts use a structural Go assertion, so a full six-method source impostor could pass where Java requires a nominal check. An additive independent challenge is being captured; the original workflow is unchanged. Full Gson and the campaign round remain unaccepted.

The canonical String integration passes its original16 JVM/Go cases and10 race cases, plus the runtime74 gate. Its native ancestry gate then exposed missing canonical inherited Object.toString adaptation. The original failing fixture and all four unrun later stages remain recorded. A new source-default adapter candidate is prepared; no acceptance is inferred from preparation.

Further focused results: general Go-keyword package lowering passes all three import/binding controls; inherited Entry passes its original three controls but has a separate binder-shadow gap; Duration/Instant nominal tests pass without further runtime changes. DateFormat now has an actual JVM-valid generated-build failure and a private candidate. Throwable construction and direct Class.getName now pass the full11 focused controls; method-reference return adaptation is a separate active followup.

Native map-view review led to two additional stateful applications for callback order and raw-key storage. All18 JVM runs passed; their Go behavior is still pending. These prerequisites also gate broad source/native Map and Set write interoperability.

The literal-cache optimization remains unmeasured. Its original compiler baseline fails to build a valid challenge because a local uint16 name shadows the generated literal element type. That semantic prerequisite must be repaired for both baseline and optimized code before comparing performance. No Java performance advantage is claimed.

The coordinator goal remains active. At most three isolated correctness lanes are assigned, and performance measurements remain exclusive. The progress50 portable evidence bundle is being assembled by sol_concurrent_v2; until sealed, campaign-state.json records the exact private artifact locations and known Git checkpoints. Checkpoint18 remains the latest production compatibility acceptance.
