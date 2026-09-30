# Round 09 queue rebase: prepared, not executed

The isolated queue patch remains source-compatible with the completed round 09 worktree. The entire current `stdjava/reference_arrays.go` is byte-for-byte identical to the earlier frozen baseline, not just the changed loops. Applying the same two textual substitutions produces the exact earlier candidate file and patch hashes. This is source inspection and preparation evidence; current-branch tests and performance observations have **not** been run.

No build, test, benchmark, Git command, or production edit was performed during the root gate hold. The candidate is ready for validation when the parent explicitly releases it.

## Provenance

[prepared.json](prepared.json) records the source capture time, complete copied-file SHA-256 manifest, frozen validation-input hashes, temporary paths, and baseline/candidate manifest hashes. This is a content-addressed worktree capture; no commit identity is inferred. Files were hashed before copying, after copying, and again at the source to detect changes during capture.

Prepared baseline and candidate: `/var/folders/c1/bq3rcd9d3y54bw8n_fp4x0sr0000gn/T/java2go-queue-round09-gjh45x4m/{baseline,candidate}`. Both include the compiler/runtime/test sources required by the prepared commands; neither includes Git state. Only `stdjava/reference_arrays.go` differs between them. The input source and prior harnesses are copied separately and hashed so later live changes cannot silently alter validation.

| Artifact | SHA-256 |
|---|---|
| Captured source manifest | `11b843463f5360fe14507dda37268aea57bf93f680e162345409d59b0be4dca0` |
| Baseline queue/runtime file | `88f01dba105053ddae561a4141d695e85b3c7324ea3ca0952d96c779fddbf3a9` |
| Rebased candidate file | `66e84bf35a711e0793b9b2a897d813840d8b2df20d0be0781ca9aef9df60b09d` |
| Identical zero-context patch | `17308a763a802719561dfec04fe4ff3737f53805b5fbf65dc2415f6aa0d3870e` |

[prepare.py](prepare.py) can make another read/copy/hash-only snapshot if the parent requests a newer capture. The current [queue.patch](queue.patch) was prepared by guarded substitutions and byte comparison, without invoking patch/Git or tests. A later Git-based application would require `git apply --unidiff-zero`; no live application is authorized here.

## Compatibility review

**DateValue and SQL identity.** `stdjava/date.go` now represents the util.Date reference protocol as `DateValue`; SQL Date/Time/Timestamp remain distinct pointer values implementing that protocol and carrying their own nominal type IDs. SQL subclasses register their parent as util.Date, which registers Object and Serializable/Cloneable/Comparable edges. Queue traversal therefore sees new nominal edges but has the same semantics. The patch does not alter interface conversion, receiver identity, millis/nanos storage, equality, null validation, or DateValue dispatch. `ObjectView[DateValue]` still checks nominal assignability before a direct Go assertion; concrete SQL casts still check the requested nominal type before the concrete assertion. The candidate must not recover an embedded `*Date` subobject in place of the original SQL receiver. No such conversion is introduced.

**Canonical generic aliases.** `canonical_generic_layout.go` emits parameterized aliases to one physical erased type when proven eligible. This removes representation differences without changing Java's nominal type checks. `ObjectView` still performs those checks before accepting a structural Go assertion; the queue patch does not move that assertion earlier. For erased-descriptor recovery, the fallback candidate list retains exact BFS order, including superclass-before-interface and duplicate/cycle suppression. Identity remains the existing pointer/shared ObjectInfo, never a new wrapper or cache entry. The round 09 factory, raw-pollution, bounded Date storage, and method-reference consumer tests are included in the deferred commands because they make incorrect early/late casts observable.

**Source casts.** Current `expression.go` routes source hierarchy casts through `ObjectView` at the relevant lowering boundaries. `ObjectView`, `ObjectDynamicType`, `JavaReferenceEqual`, and the provider/constructor lifecycle are untouched by the candidate. Receiver evaluation order and single evaluation remain compiler responsibilities; the existing source-downcast/null/bad-cast oracle is included to catch a regression at this boundary.

**Monitor and GC contracts.** Current `monitor_observation.go` uses lookup-only canonical identity. The queue patch does not touch monitor identity, the logical Execution owner, acquisition, wait/notify, or locks. Both hierarchy traversals retain their original registry read lock for the whole walk, and registration replacement retains its write lock. There is no new cache/global root, identity provider, or per-instance allocation. The queue's consumed prefix stays local until return; large graphs can retain a larger temporary buffer than a frontier-only queue, so there is no universal peak-memory guarantee. Actual-acquisition monitor registry lifetime remains outside this optimization.

## Deferred validation commands

[VALIDATE_AFTER_RELEASE.sh](VALIDATE_AFTER_RELEASE.sh) contains the exact ordered commands. It refuses to proceed without `QUEUE_VALIDATION_RELEASED=1`. After explicit parent release, run:

```sh
QUEUE_VALIDATION_RELEASED=1 bash performance/runtime/object_identity_candidate/rebase09/VALIDATE_AFTER_RELEASE.sh
```

The script first verifies prepared content hashes, then performs baseline allocation-red/candidate-green checks, both complete runtime suites and race suites, and focused current-compiler source oracles for SQL/DateValue, canonical generic alias factories/storage, null/source casts, and all monitor campaign tests. It then builds one prepared baseline compiler, generates each workload once, and builds identical generated application bytes against the two runtime variants. The same JDK 21 source observations and 18 process parity/heap comparisons are preserved, followed by three fresh allocation processes per variant. JVM/Go parameters and explicit JDK path match the original study. No timing metric is collected.

The prepared shell script has not been executed. Any failure must remain visible; a baseline red result must contain the expected allocation assertion, not merely a build failure. Expected allocation improvements from the earlier snapshot are hypotheses on this current source until the deferred run passes. Updated registered-type counts may differ from the earlier 99, so validation should require stable per-run counts rather than that old absolute constant. No speedup or memory superiority over Java is asserted.
