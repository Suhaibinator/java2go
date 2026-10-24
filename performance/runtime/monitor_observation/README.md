# Thread.holdsLock / Writer monitor observation audit

**Finding:** observing an unheld object currently creates a permanent monitor record and retains the object graph, even though `Thread.holdsLock` returns false. An isolated lookup-only candidate removes this observation-only allocation/retention while preserving the existing logical-execution ownership check and all actual synchronization. Real monitor acquisitions, including Writer defaults, still retain objects through the existing registry.

No live production/compiler/challenge edits, Git actions, campaign gate additions, or wallclock performance claims. Runtime suites, targeted diagnostics and matching JVM/generated-Go programs ran against temporary baseline/candidate runtime copies. The reviewable candidate is [lookup-only.patch](lookup-only.patch); it has not been applied to the live runtime.

## Mechanism and measured evidence

`stdjava/monitor_observation.go:8` calls `monitorRecord(value)`. `stdjava/concurrent.go` computes an identity, takes `monitorsMu`, and creates both a `monitor` and its `sync.Cond` on a miss. The global `monitors` map never removes them. Comparable keys hold the queried object strongly; `monitor.anchor` independently holds it. Slice/map identities use an address but still retain the underlying object through the anchor. This is process-lifetime retention, not merely lifetime-of-live-object monitor storage.

Isolated runtime diagnostics:

| Operation | Baseline | Lookup-only candidate |
|---|---:|---:|
| Registry records added by 32 false queries on new arrays | 32 | 0 |
| Allocations/query for first observation of preallocated fresh object | 2 | 0 |
| Allocations/query when repeatedly observing one already-observed object | 0 | 0 |
| Registry growth from repeated queries on the same object | 0 | 0 |

The allocation probe uses `testing.AllocsPerRun(32, ...)` with all 33 objects preallocated (including its warmup invocation), so object construction is excluded. Counts are Go-only diagnostics, not JVM allocation measurements; occasional map growth at other table sizes may add allocation beyond the two observed record/condition allocations. Registry-length evidence is independent of allocator sampling.

The shared [Java application](source/MonitorObservation.java) was compiled with JDK21 and strictly transpiled once; identical generated source was linked against baseline and candidate snapshots. Every fresh process executes three batches of 32 objects × 128 KiB payload = 4 MiB per batch, dropping all application references after each batch. Each snapshot follows two forced collections. Three independent process triplets per scenario matched semantic stdout exactly.

| Scenario / post-GC heap delta from process baseline | Batch 1 | Batch 2 | Batch 3 |
|---|---:|---:|---:|
| Unheld-array queries, baseline Go | 4,204,264 B | 8,408,808 B | 12,608,744 B |
| Unheld-array queries, candidate Go | 16 B | 32 B | 32 B |
| Unheld-array queries, Java | -200,848 B | -169,288 B | -173,912 B |
| Self-lock Writer writes, baseline Go | 4,338,424 B | 8,677,112 B | 13,011,192 B |
| Self-lock Writer writes, candidate Go | 4,338,424 B | 8,677,112 B | 13,011,240 B |
| Self-lock Writer writes, Java | -200,840 B | -169,280 B | -173,904 B |

Cells are medians of three fresh processes. Candidate observation-only final deltas range 16–144 B. The negative Java deltas reflect startup/ownership-check/formatting objects becoming collectible after the baseline; they do **not** represent negative allocation. The useful comparison is growth with additional dropped 4 MiB batches: Go baseline retains each batch, Java and the lookup-only candidate do not show payload-proportional growth. Heap accounting/collector and object layouts differ, so these are lifetime observations, not exact cross-runtime per-object sizes or whole-process RSS comparisons.

Writer scenario: a source `Writer` subclass owns a 128 KiB array, uses inherited `write(int)`, asserts `Thread.holdsLock(this)` inside its `write(char[],off,len)` override, and is then closed/dropped. Java/generated checksums match. The lookup-only candidate correctly leaves this path's real monitor acquisition intact, so its retention remains. Closing this custom Writer is a no-op on both sides; it cannot remove the registry's root.

Exact semantic checks include:

```text
ownership=false:true:true:false:false:NPE
```

This covers outside ownership, owning execution, reentrant ownership, another Java thread observing false while the first owns the lock, false after exit, and plain Java null rejection. Array-query checksum lines are 496/528/560; Writer lines are 2576/2576/2576. All eighteen Java/baseline/candidate scenario-process outputs match. Each process has a 30-second hard timeout; elapsed times are not used as performance evidence under campaign load.

## Candidate correctness argument

The patch preserves `requireExecution` and `requireNonNullMonitorReference`, then uses the existing identity function and `monitorsMu` for a map lookup **without insertion**. If absent, it returns false. Otherwise it releases the registry lock, takes the existing record's `mu`, and checks `owner == execution && depth > 0` exactly as before.

- A missing record means no registered ownership exists at that lookup point. A later competing entry can be ordered after the observation; returning false is valid. A logical Java execution cannot already own a monitor that was never registered.
- Existing records never disappear today, so the pointer returned under the registry lock remains valid when `monitor.mu` is subsequently acquired. If registry lifetime management changes, this proof must be revisited.
- All map access remains synchronized; ownership/depth reads remain synchronized. The candidate does not acquire `legacyMu`, block until another owner's entire synchronized body finishes, or infer Java identity from OS/goroutine identity.
- Monitor entry/exit, reentrant depth, wait release/reacquisition, notification, legacy interoperability, and Writer callback dispatch are untouched. No required synchronization is removed. Returning a false ownership observation is not a replacement for a Java monitor acquire or its happens-before guarantees.
- Full copied `stdjava` suites pass for baseline/candidate. A targeted no-insertion assertion fails baseline and passes candidate. Existing-owner/other-execution checks and the shared Java ownership program pass.

Acceptance recommendation: review this narrow change separately from monitor lifetime redesign; preserve all lookup/owner locking and validation, add the no-registration regression to the runtime owner's suite, and rerun the Writer monitor/buffer/virtual-default JVM oracles plus existing monitor wait/reentrancy/race coverage. Do not claim all monitor identity semantics are repaired by this patch.

## Identity and null limitations found during inspection

Two diagnostic observations are present in **both** baseline and candidate:

1. Two carrier views sharing one `ObjectInfo` compare equal under `JavaReferenceEqual`, but entering with one and querying the other returns false. `monitorIdentityFor` uses the directly comparable Go view rather than the canonical `ObjectInfo`. This is a mismatch between existing runtime identity protocols, not introduced by lookup-only observation. The additional source-level fixture below confirms a generated base/derived alias failure as well. Its emitted simple hierarchy uses separate base/derived pointers without shared ObjectInfo, so adding an ObjectInfo fast path only in the runtime would not cover every current source object. Changing only the holdsLock key would also be wrong: acquisition/query/wait/notify must use the same canonical key.
2. Passing the runtime `NullString()` sentinel directly to holdsLock does not throw NPE. `nilMonitorReference` handles Go nil/reference kinds, not the string sentinel used elsewhere by `javaReferenceIsNull`. The original ownership oracle above verifies plain null only; the additional nullable-String source oracle below reproduces the gap without a cast. Its generated local is widened through StringReferenceValue but still reaches holdsLock as a sentinel-backed string. The candidate preserves current validation and does not silently broaden its scope.

These limitations are reported explicitly rather than being hidden by a passing narrow ownership oracle. They are especially relevant before changing monitor representation/lifetime.

### Source-level follow-up: both limits reproduce

[MonitorEdges.java](edges/MonitorEdges.java) uses an ordinary `ObservationBase base = derived` assignment and `String absent = null`; no manual casts or generated edits. Its four output fields are: inherited base method synchronized on `this` querying the derived alias; query of base while synchronized on base; query of derived in that same region; query of absent String.

```text
JVM:          true:true:true:NPE
Baseline Go:  false:true:false:returned-false
Candidate Go: false:true:false:returned-false
```

This is deliberately recorded as a **failing semantic diagnostic**, not a passing optimization/campaign gate. The candidate agrees with baseline and does not fix either mismatch. Generated output shows monitor entry using the base subobject pointer while the alias query uses the derived pointer; nullable String reaches the observation helper through the existing string-reference conversion. Root has the evidence for checkpointed TDD ownership; no live fix was made.

An initial supplementary `base == derived` assertion encountered a separate generated build error comparing incompatible Go pointer types. The initial source, generated file and error are preserved as `edge-initial-*` raw artifacts. Removing only that assertion from this owned diagnostic lets the requested monitor/nullable-String behavior execute; the alias is still established by the unchanged base assignment. This is not a change to any frozen challenge fixture.

Reproduce after the main runner, substituting its output directory:

```sh
python3 performance/runtime/monitor_observation/check_edges.py .campaign/performance/monitor-observation-20260927T232210
```

Exact outputs and commands are saved as `edges-results.json` and `edges-commands.json` beside the primary results; `edge-generated/` contains the unchanged transpiler output.

## Writer paths and general follow-on candidates

`WriterWriteDefaultExecution` for an integer and `WriterWriteStringRangeDefaultExecution` obtain `WriterState`, enter `state.Lock` with the caller's `Execution`, defer exit, and invoke the virtual char-array writer with that same token. This is necessary for callback `holdsLock`, reentrant synchronization and exception-safe release. Direct char-array overload dispatch does not acquire a new monitor by itself; do not add/remove locks across overloads without the JVM contract tests.

Default Writer lock is the receiver. Its global monitor key/anchor consequently roots the Writer, its payload/closure fields, and its cached 1024-element `PrimitiveArray[rune]` write buffer. The 4-byte Go rune storage and wrapper add storage relative to a Java char array, but no representation redesign was measured. An externally supplied independent lock usually roots that lock, not necessarily the Writer graph; if the lock references the Writer or a larger graph, that graph is still retained. Lifetime depends on reachability, not class name.

The proper general fix remains owner-attached monitor state on canonical object identity, with a complete strategy for arrays, runtime/external references and class objects. Object-owned cycles can be collected when no global root exists. **Do not** remove anchors, store bare addresses without lifetime protection, or delete records on final unlock: waiters/contenders may already hold an old record, creating split locks or address-reuse bugs. Pooling per-Writer buffers globally also changes callback-visible buffer identity and concurrency, which existing Writer tests observe.

`EnsureWriterBase` takes a global mutex even when a writer's base already exists. That is a separate potential contention cost, not measured here. An unlocked `if *slot != nil` check is not automatically safe when other goroutines can perform lazy initialization. A safe approach would require eagerly initialized immutable base state before publication, or an atomic publication protocol consistent with constructor/virtual-call behavior. Preserve explicit supplied locks, constructor ordering and callback dispatch before measuring a faster path.

## Reproduction / evidence

```sh
python3 performance/runtime/monitor_observation/run.py
```

Coordinate a brief allocation/heap window first. The runner creates temporary runtime snapshots, applies `lookup-only.patch` there, runs copied runtime/diagnostic tests, strictly transpiles the source once, and builds both variants. It never edits live production files. Generated source, raw commands, test outputs and per-process memory records are saved beneath `.campaign/performance/monitor-observation-<timestamp>/`.

This run: `.campaign/performance/monitor-observation-20260927T232210/results.json`. Extra fresh-object allocation diagnostics are `baseline-allocation-diagnostic.stdout` and `candidate-allocation-diagnostic.stdout`; the current runner includes that diagnostic in its normal output. Temporary modules: `/var/folders/c1/bq3rcd9d3y54bw8n_fp4x0sr0000gn/T/java2go-monitor-observation-x9v3toe_/{baseline,candidate}`. Production source fingerprints are recorded in results.

JDK21 explicitly `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`, SerialGC, `-XX:ActiveProcessorCount=2 -Xms32m -Xmx256m`; Go `GOMAXPROCS=2 GOMEMLIMIT=256MiB GOGC=100`. Those are matched requested CPU/nominal memory envelopes, not identical collector policies or hard process-RSS limits. No speed comparison or warmed-JVM throughput claim is made.
