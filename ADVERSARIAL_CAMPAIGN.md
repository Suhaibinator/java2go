# Continuous adversarial campaign

Status: round 01 accepted; round 02 remains active. Checkpoint10 verifies five applications with 45 exact JVM/Go observations and 100 additional race-enabled stress runs. Full unit, strict application parity, other end-to-end, fuzz replay, repository build and exact golangci-lint 2.14.0 gates pass on the immutable snapshot. Eight known-open fuzz cases remain skipped, identical to checkpoint07 and fewer than the original baseline’s fourteen; no added failures or skips are accepted.

The unchanged Gson application has no accepted Go observations. Strict transpilation completes in 94.991 seconds under the unchanged 300-second limit. Go compilation still fails on missing AbstractMap, AbstractSet, BigInteger, BigDecimal and ConcurrentMap support and an omitted interface-nested enum. Complete selected dependency sources remain frozen; no dependency behavior is replaced or delegated to the JVM.

Original CI lint findings are repaired. Exact golangci-lint 2.14.0 passes the recorded snapshot; later source edits require another lint gate. Verified arithmetic and performance-evidence milestones are committed and pushed. `campaign-state.json` records immutable fingerprints, preflight failures, focused repairs, run handles and exact resume points. Base revision: `c1f1ae45cfe8d5465425efc562a54580add4f3ea`.

Three Sol agents own independent Java applications and JVM-derived oracles. Three Astra agents own compiler, runtime, and dependency/harness implementation respectively. Two additional dedicated Astra agents investigate generated-code and runtime performance; they test isolated optimization candidates and do not change live production code or challenge oracles. The coordinator independently verifies every promotion. No challenge or upstream implementation may be weakened to make translation pass.

## Execution contract

Use the installed JDK 21 explicitly; seeds 17, 41, 97 each run three times before freezing. Dependencies and complete selected implementation source graphs are locked and translated, never replaced by handwritten library behavior or a JVM bridge. Unsupported calls, timeouts, compilation failures and changed observations are failures. Java package cycles must be handled by the compiler rather than source reorganization.

Work happens in the managed `adversarial-java` checkout. Existing user changes remain in the original checkout. Agents own disjoint files; coordinate before shared edits. Transpiler processes and scratch directories are isolated because symbol tables are global. Commit and push verified milestones to the authorized origin branch; publish no unrelated changes.

## Teams

| Agent | Model | Ownership |
| --- | --- | --- |
| sol_business | gpt-6-sol, high | testfiles/campaign/round*/business |
| sol_data | gpt-6-sol, high | testfiles/campaign/round*/data |
| sol_concurrent | gpt-6-sol, high | testfiles/campaign/round*/concurrent |
| astra_compiler | gpt-6-astra, high | Compiler, symbols, parsing, project lowering |
| astra_runtime | gpt-6-astra, high | stdjava runtime and coordinated lowering |
| astra_build | gpt-6-astra, high | Dependency bootstrap, campaign runner and harness |
| astra_lint | gpt-6-astra, high | Exact CI lint remediation and regression checks |
| astra_perf_codegen | gpt-6-astra, high | CPU/allocation audit and measurement under performance/codegen |
| astra_perf_runtime | gpt-6-astra, high | Runtime/concurrency/memory audit under performance/runtime |

## State and resumption

`campaign-state.json` records active challenges, dependency pins, baseline, and gates. `.campaign/` contains ignored downloaded artifacts and run logs; locks and fixture sources are tracked. Rebuild the compiler and rerun affected gates whenever implementation changes. Accepted challenges remain permanent regressions. After a passing round, Sol creates fresh challenges. Continue until explicitly stopped or execution/usage limits intervene; never equate corpus parity with universal Java support.

## Verification commands

Run a frozen application with full concurrency acceptance using:

```sh
JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home \
TMPDIR=/private/tmp go run ./cmd/javacampaign \
  -fixture testfiles/campaign/round01/concurrent -race -stress-runs 20
```

The flags apply to any fixture. The runner builds all generated packages and the application with race instrumentation, runs the nine normal pairs, then cycles frozen seeds through twenty additional independent executions against the validated JVM oracle. Every result is recorded, and source/implementation fingerprints must remain unchanged. The tagged campaign CI gate requests both flags for every fixture. Uninstrumented runs remain available for diagnosing the first build failure and do not satisfy concurrency acceptance.

The performance reports distinguish observed regressions, isolated optimization candidates, and demonstrated application speedups. Currently there is no demonstrated whole-application speedup. Source-level compatibility remains the round's promotion gate.

## Milestone checkpoints

Commit and push meaningful milestones to `origin/codex/adversarial-campaign`, as explicitly requested by the user. Keep incomplete repair patches uncommitted when recording a stable checkpoint. A checkpoint may record an active failing challenge; it must state that failure clearly and must not be labeled a completed round. Never force-push milestone history.

### Checkpoint06 follow-up

Five accumulated applications passed 45 JVM/Go pairs and 100 stress runs. Exact lint 2.14.0, strict application parity, other end-to-end tests and fuzz replay passed. The unit gate found an affine helper naming regression, now repaired against the unchanged test. Two concurrent full Gson runs timed out during conversion; a quiet instrumented replay completed in 96.810 seconds under the same 300-second bound. Checkpoint07 reruns the official Gson harness quietly before fresh historical verification. Neither timeout is accepted as parity.


### Verification scheduling

Checkpoint07 exact lint is clean. Its first full unit and strict application-parity gates timed out, including a Java oracle; those runs are failures. Sequential reruns under the original limits now pass: the full unit compiler package took 447.746 seconds and strict parity took 653.655 seconds; other end-to-end, fuzz and all five historical dependency applications also passed. Input hashes remained unchanged. This verifies checkpoint07, while full Gson still fails and round02 remains active. Candidate08 and its focused repairs remain separate from this verified implementation.


### Checkpoint09 repair status

The exact 2.14.0 lint gate passed after removing one unused helper. The immutable full unit gate failed with eight new regressions; later gates did not run. Boxed generic consumption and scope-aware local identifier repairs now pass focused race checks, including cast timing, null unboxing, resource cleanup and generated implicit types. These changes are integrated but not yet a verified milestone. Checkpoint07 (`032d3550`) remains the last fully verified and pushed checkpoint.


### Checkpoint10

Connected generic families now preserve shared storage, nominal casts, boxed consumption, local interface implementations and varargs SAM array checks across the exercised contracts. SQL date/time types preserve their canonical owner and util.Date relationship, including bounded generic method references. Local identifier hygiene respects body requirements and renamed resource cleanup. Passing the render context through execution-name generation prevents repeated family rediscovery.

Independent verification checked raw streams, declared files, frozen source/dependency/resource hashes, all nine repeated JVM oracles per application, translated-source versus binary dependency oracles, and twenty race-enabled stress runs per application. All historical observations matched. A verbose original/checkpoint07 fuzz audit confirms the eight remaining skips are pre-existing. Gson’s Go compilation failure remains an explicit blocker, so this is a checkpoint, not round02 acceptance. Private nested-enum and big-number TDD work continues after this milestone; unmeasured hygiene caching and runtime allocation candidates remain isolated.
