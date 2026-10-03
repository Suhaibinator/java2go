# Runtime performance audit — 2026-09-27

Scope: runtime, concurrency, memory, long-lived services. Read against worktree HEAD `c1f1ae45cfe8d5465425efc562a54580add4f3ea` plus concurrently evolving campaign changes. This directory contains diagnostic tooling and recommendations only; it does not change production runtime/compiler code. References describe the inspected snapshot and may move as the correctness agent edits files.

**There is no measured Java-versus-generated-Go speed win in this report.** Retained heap and repeated callbacks below are direct Go observations. Expected throughput/latency improvements are hypotheses pending parity-verified, isolated measurements. Coordinate sustained runs with the campaign root; other agents currently compete for CPU.

## Findings ranked for implementation

| Rank | Candidate | Evidence | Expected benefit (hypothesis) | Risk / prerequisite |
|---|---|---|---|---|
| 1 | Atomic successful-class-initialization fast path | `stdjava/class_initialization.go:47-102`: every `Ensure` locks, including permanent initialized state; generated active uses call these coordinators | Remove shared mutex acquisition on hot static calls/fields; permit multicore scaling of otherwise independent work | Medium: publish success only after all initializer writes; keep recursive-initializing, failure, wakeup, nil validation, and superclass semantics unchanged |
| 2 | Object-owned monitor identity and lifetime | `stdjava/concurrent.go:531-618`: global mutex lookup plus permanent comparable key and `anchor`; retention probe retains all payloads after exit | Bound live heap by live objects; reduce GC scan work and eliminate global lookup contention between unrelated generated objects | High: canonical generated `ObjectInfo` ownership, array/runtime/external-reference strategy, alias/legacy interoperability, wait/notify and contenders must be solved together |
| 3 | Clear vacated list slots | `stdjava/list.go:69-72`: truncation via append never clears last pointer slot; probe observes 32 stale refs after 32 tail removals | Immediate release of removed object graphs while live list capacity remains; lower steady heap/GC work | Low: preserve removal order/return/exception behavior and storage semantics; keep undefined external retention of `Slice()` out of the contract |
| 4 | Bucket-local CHM validation, then sharding | `stdjava/concurrent.go:93-146`: single global version invalidates all writers; immutable bucket copies before publish | Less unrelated callback repetition and discarded allocation; improve disjoint-key write scaling | Medium/high: callbacks stay outside map locks, same-key updates remain linearizable; bucket deletion/recreation ABA prevention; size and snapshot semantics |
| 5 | Reusable executor queue storage | `stdjava/futures.go:164-188`: front slicing permanently consumes reusable capacity, although popped closure is correctly zeroed | Fewer backing-array allocations for sustained enqueue/dequeue; bounded retained queue capacity after bursts | Low/medium: FIFO dequeue, unbounded submission, shutdown rejection/drain, cancellations and condition signaling unchanged |
| 6 | Cache execution companion resolution or emit typed dispatch | `stdjava/object_methods.go:96-132`: each hash/equality walks all reflected methods, allocates call parameters and calls reflection | Reduce collection key-operation CPU and allocation without removing logical execution propagation | Medium: collision-safe names, Object overload selection, inherited dynamic receiver, nil arguments and user exceptions; cache method metadata, never bound receiver values |
| 7 | Lazy/fused stream evaluation after semantic repair | `stdjava/stream.go`: source copy + fresh slice for most stages + terminal copy; explicitly eager approximation | Reduce O(stages × elements) live intermediates and allocations; short-circuit useful work | High: current eager behavior differs from Java; first implement Java operation order, short-circuit, source binding, consumed-state and exception timing |
| 8 | Type-assignability metadata cache | `stdjava/reference_arrays.go:190-279`: repeated registry RLock + BFS seen map/queue | Reduce array-store/cast overhead for repeated type pairs | Medium: registration can mutate; use generation invalidation or explicit registry freeze; negative caches cannot survive later registration blindly |

Rank balances reach, effort and confidence; rank 2 is particularly important for server viability, but is not a safe quick patch. Rank 3 is the most mechanically narrow first fix. The runtime correctness agent has been notified of ranks 2–4.

### Class initialization implementation boundary

A separate atomic success flag is simpler to audit than mixing atomic and non-atomic accesses to the existing state. After validating receiver/execution, load the flag and return on success. Slow paths keep the current lock/state machine. Only the successfully initializing execution stores true, after its initializer body and required class dependencies complete. Retain lock-protected error cause creation. Do not replace `Ensure` with `sync.Once`: same-execution recursion and permanent erroneous state have distinct behavior. Keep the zero-value coordinator usable. Validate publication with concurrent readers of initialized non-atomic fields under `-race`, recursively initialized classes, thrown Error vs non-Error, and repeated failed active uses.

### Monitor lifetime / contention boundary

The registry comment says monitor lifetime follows object lifetime, but its strong root actually extends object lifetime to process lifetime. Both `monitorIdentity.comparable` and `monitor.anchor` can retain objects. An anchor can transitively retain a whole request, cache, graph, or captured closure. Merely deleting `anchor` does not release pointer keys. Merely deleting an idle registry record can create two monitors when another goroutine already holds a record or is waiting; address reuse also matters.

Generated objects already share `ObjectInfo` across hierarchy views (`reference_arrays.go:282-350`). An optional lazily installed monitor there is a promising owner-controlled design: the object may own monitor state without a global root, and all Java views share it. A Go object/closure cycle is not by itself a global root. This requires extending canonical identity to runtime objects/arrays; plain primitive-array slices and external Go values do not all provide the same carrier. Class monitors may safely be long-lived per class but should share identity with the corresponding Class object when supported.

Retain explicit execution ownership, reentrant depth, atomic wait release/reacquire, wait-set lifetime, notification ownership exceptions, and exclusion between legacy and execution-aware entrypoints. Distinct-lock benchmark must also verify same-lock exclusion and cross-view identity before any timing. Do not pool execution tokens across simultaneously live Java executions. A guard allocation should only be optimized after escape analysis shows it escapes; a returned pointer does not by itself prove a heap allocation at every inlined call.

### Concurrent map boundary

A single global version causes a disjoint-key update to retry, even though its collision bucket is unchanged. The deterministic probe below forces exactly this interleaving without time-based sleeps. This is additional CPU/allocation, and repeated user callbacks also deserve semantic scrutiny before performance claims.

First compare immutable bucket identities/generations rather than a global mutation version. A deleted and recreated bucket must not accidentally reuse a validating generation. Next consider shards selected by Java hash, each retaining immutable bucket snapshots and out-of-lock callbacks. A per-bucket or shard mutex held during user equality/hash callbacks can deadlock reentrant operations and is not an acceptable shortcut. Snapshots and size must remain coherent to the extent the supported runtime promises. Colliding keys still need careful handling; native Go map equality is not Java equality, and pointer identity cannot replace virtual `equals`.

### Executors and logical execution

`ExecutorService.worker` already creates one `Execution` per worker and refreshes it after an uncaught `execute` exception. It does not create a token per successful task. `submit` captures exceptions in the Future and uses the worker token. Preserve this lifecycle and current cancellation/publication behavior. A circular queue or head-index buffer with zeroing/periodic compaction can reuse capacity under the same mutex. Drain/reset policies should avoid retaining an enormous empty backing array forever; popped closures are already nilled and should stay so. Do not replace the unbounded queue with a blocking bounded channel: submission/deadlock and shutdown behavior would change.

`Thread` retains its `run` target after termination; investigate releasing a private post-termination target only after parity around later direct `run()` calls, subclass behavior and retention is established. It is not an unconditional safe recommendation. Existing goroutines are expected to live while an unshut fixed pool lives; benchmark pool shutdown and termination, rather than reporting those workers as a leak. Non-daemon JVM lifetime versus Go main-return lifetime requires semantic qualification in application benchmarks.

### Collections, strings and streams

Hash-map removal currently linearly locates the insertion-order record and shifts `entries`; TreeMap insertion/removal also shifts the sorted slice. Large mutable maps deserve profiling before promising throughput. A linked insertion-order record/indexed structure could avoid shifts for hash maps, at a locality and memory cost. Keep Java hash/equals, comparator invocation behavior, deterministic supported order and view contracts. `Map.Remove` already clears its vacated pointer slots, unlike the inspected `List.RemoveAt`.

String representation and UTF-16 observation are under active correctness repair. At audit time, repaired `StringCharAt` materializes `StringChars` per call; repeated charAt loops can allocate a full UTF-16 view per index. The CPU specialist owns concrete string-loop suggestions. Avoid a process-global string-to-UTF16 cache: it would retain arbitrary service input indefinitely. Prefer a proven loop-local immutable view or allocation-free decoder/ASCII fast path, preserving UTF-16 units, surrogate behavior, null/bounds timing. `StringBuilder` uses rune storage and advertises incomplete surrogate/StringBuffer semantics; correctness must precede performance claims. A faster incorrect Unicode/string synchronization workload is not evidence of winning.

Eager stream copies are real overhead, but replacing them with aliasing alone may change source mutation and exception behavior. Prefer a proper single-use lazy pull/push pipeline, then fuse compatible stateless stages; materialize where sorting/distinct semantics require it. Preserve element encounter order, null rules, all observed callbacks, terminal short-circuit and cancellation. Restrict an initial fused codegen optimization to proven semantics rather than assuming user callbacks are pure. Do not count skipping required callbacks as a win.

## Reproduced lightweight evidence

Host: Darwin arm64, Go `go1.27.1`. Each heap mode ran in a fresh process with `GOMAXPROCS=1`, 32 payloads × 131,072 bytes = 4,194,304 payload bytes, two forced collections before and after. No sustained benchmark ran. Heap deltas include allocator/runtime bookkeeping and are not exact per-object size models.

| Diagnostic | HeapAlloc delta after GC | HeapObjects delta | Stale references beyond list length |
|---|---:|---:|---:|
| Baseline allocate/drop | 16 B | 1 | 0 |
| Enter + exit each object's monitor | 4,204,072 B | 131 | 0 |
| Remove every list element from tail | 4,195,352 B | 66 | 32 |
| List.Clear control | 40 B | 2 | 0 |

The object payloads are not kept by the diagnostic after the allocation function returns. The removed list itself is deliberately kept live. `Slice()[:cap]` inspection is a diagnostic of runtime storage, not a claim that generated Java can observe that capacity. The monitor scenario has no ongoing owners or waiters when GC runs.

Deterministic CHM result: `equal_calls_for_one_update=2`, `final_key1=11`, `final_key2=20`, `size=2`. Key 1's first equality callback blocks; key 2 is inserted in a different hash bucket; key 1 resumes and retries. This demonstrates unrelated invalidation, not a timed slowdown or Java behavior parity.

Reproduce from repository root:

```sh
go build -o /private/tmp/java2go-runtime-retention-probe ./performance/runtime
for mode in baseline monitor list-remove list-clear; do
  GOMAXPROCS=1 /private/tmp/java2go-runtime-retention-probe -mode "$mode"
done
go run ./performance/runtime/cmd/chmretry
```

Optional one-process profile (sampling is not exact allocation counting):

```sh
GOMAXPROCS=1 /private/tmp/java2go-runtime-retention-probe -mode monitor -heap-profile /private/tmp/java2go-monitor.heap
go tool pprof -top -inuse_space /private/tmp/java2go-runtime-retention-probe /private/tmp/java2go-monitor.heap
```

## Scheduled Java/generated-Go validation matrix

Before any sustained timing, request a free measurement window from root. Use the same Java sources transpiled by the current compiler and compare every deterministic checksum/output/exception trace first. Keep compiler/runtime revision, generated-source hash, commands and raw trial output. A manually rewritten Go approximation is not a generated-code benchmark.

| Workload | Inputs and observable parity | What to measure |
|---|---|---|
| Hot class active uses | Initialized class with static getter/field; multiple workers each accumulate identical seeded values; separate initializer publication/recursive/failure fixtures | Warm ns/op and 1/2/4-worker scaling, Go mutex profile, allocations |
| Independent request locks | One distinct object lock per request with 128KiB payload; same-lock control; base/derived aliases; deterministic total | GC-retained heap vs completed request count, peak RSS, allocation bytes, throughput and GC pause distribution |
| Map disjoint writers | 1/2/4 workers each own disjoint fixed key ranges; controlled 0/1/25% collision distributions; matching final map/checksum; reentrant and throwing equality as separate semantic fixtures | Throughput, callback counts, bytes/op, mutex/block/CPU profiles; same-key histories independently validate atomicity |
| Live list churn | Fixed live list, insert/remove large payloads, compare return values/order; tail and middle removal | Post-GC live heap plateau, allocation and GC work; avoid timing GC-forcing loops as service throughput |
| Fixed executor service | Same workers and seeded tasks, tiny and CPU-sized jobs, burst/steady arrivals; Future results, failure/cancel/reject/shutdown outcomes | Warm task/s, submit-to-complete p50/p95/p99, allocations, goroutine/thread count before/after termination |
| Collection virtual callbacks | Keys with custom hash/equals, inherited overrides and member-name collisions; same successful lookup checksum and exception traces | CPU profile, reflect dispatch counts, alloc/op before/after metadata cache |
| Stream/string pipeline | Stable data sizes 1Ki/64Ki/1Mi elements; ASCII/BMP/supplementary strings; pure and side-effect/short-circuit fixtures | Allocated bytes, live heap, stage/element callback counts, warmed throughput only after parity |

Use explicit JDK 21 path, never the shell's implicit Java:

```sh
export JAVA_HOME=/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home
"$JAVA_HOME/bin/java" -version
# Suggested resource envelope for a root-scheduled four-CPU trial:
# Java: "$JAVA_HOME/bin/java" -XX:ActiveProcessorCount=4 -Xms512m -Xmx512m ...
# Go:   GOMAXPROCS=4 GOMEMLIMIT=512MiB ./generated-program ...
```

A JVM `-Xmx` and Go `GOMEMLIMIT` are different controls, not identical heap semantics. Report both configured budgets plus actual live heap and process peak RSS; enforce the same total-memory envelope when comparing service capacity. Match input seed/data volume, requested concurrency, visible CPU budget, output validation, and memory envelope; record oversubscription/background work. Repeat at one CPU to distinguish synchronization from parallelism.

Separate cold process startup from sustained steady state. In a single JVM process, run warmup until throughput/compiler activity stabilize, then multiple timed epochs over fresh equal input state; consume a checksum outside timing to prevent dead-code elimination. Apply corresponding untimed setup/warmup to Go. Run at least five independent process trials after the window is clear, alternate Java/Go order, retain all samples and uncertainty. Compilation and input/output formatting do not belong in compute-only timing unless separately labeled. Do not infer a sustained win from a short startup-dominated Java invocation. CPU/mutex/block/heap profiles should run in separate diagnostic trials because instrumentation changes timings.
