# Sol progress66: coherent implementation checkpoint and active repairs

This checkpoint publishes the accumulated compiler/runtime implementation and regressions from the frozen 2392-file candidate. The 694-path delta is confined to astutil, stdjava, symbol and transpiler. It preserves 43 newer campaign runner, resource-ingestion, dependency-lock, ledger and reproducer paths. The resulting 2425-file publication manifest is `23ac5539d4d295e3d3345f619ee037dad5bba3251b0a6adf7a0c5c15f73b267b`.

Fresh coordinator execution passed `go build ./...` and `go test -race -count=1 ./campaign`, with unchanged source hashes and supervised cleanup. The original 11 compiler applications independently pass on source2392. The harder three-application gate retains a runtime mismatch: erased typed-null comparison reports false instead of the JVM true. A general compiler fix using existing reference equality is undergoing focused GREEN and fresh application verification. This is a work-in-progress implementation checkpoint, not full coherent CI or dependency acceptance.

The separate native branch `codex/adversarial-native-checkpoint19` at `4f9a0c2` includes the UTF-8 JVM oracle stream repair. Its full unit gate passed 2037 test/subtest records without failures or skips; both E2E shards passed. Baseline and candidate fuzz runs have identical 45 terminal keys: 36 test passes, eight existing skips and one package pass. Detailed skipped-child output is not retained by the original fuzz harness, so no fuzz corpus parity is claimed. Remaining native CI jobs continue; a local Python3.9 bootstrap binding failure is recorded as infrastructure, not compiler behavior.

Original Codec acceptance keeps the full application, dependency implementation, seeds and oracle observations frozen. The first coherent attempt ran 18 successful JVM observations but stopped before Go compilation because its plan bound a nonexistent source directory. The corrected transport is independently reviewed and running; neither attempt changes the acceptance contract.

Four reflection controls now have actual JDK21 observations from four compilations and 36 runs, with repeat stability. Runtime and compiler workers are deriving TDD regressions for general reflection support; full Gson Go acceptance remains pending.

Netty preparation compiled 743 unchanged implementation sources into 1573 class files. Independent audit verifies sources, tools, class hashes and cleanup. This establishes javac input readiness only; no Netty-to-Go acceptance or stage advancement is recorded.

Eighteen retained Sol roles have been reassigned to implementation, challenge authoring, independent verification and two dedicated performance tracks. Four isolated Go correctness lanes and two JVM-only lanes run concurrently. Performance measurements require exclusive execution and accepted correctness; no speedup has been measured. Astra remains stopped. Checkpoint18 remains the latest full compatibility acceptance.

Resume from live agent receipts: null/header integration and original11/hard3; original Codec56; Gson reflection RED/fix/GREEN; native remaining CI; Normalizer/casing; comparator/builder application correctness; queued fresh Commons and Netty challenges. Continue the adversarial loop, commit and push milestones, and preserve every frozen challenge and historical failure.
