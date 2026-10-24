# Round 2: latch, thread identity and concurrent service

Read-only audit of the newly added runtime paths and frozen `testfiles/campaign/round01/concurrent` application. No production edits, sustained timings, or Java/Go speed claims. CPU specialist has the first sustained measurement slot. Raw diagnostics are under ignored `.campaign/performance/`; reproducible tooling is tracked here.

## What the concrete fixture establishes

The application is a correctness workload: nine submissions across a one-worker pool and a three-worker pool, seven successful tasks, one failing task, one queued cancellation. It creates/shuts down both pools each process. Three latches per phase coordinate entered/release/done. Shared synchronized `Ledger` methods guard two maps, transition counts, and worker ownership. Codec work is real upstream `Hex` and UTF-8 round trips; results are collected in submission order. Seeds 17/41/97 each have exact frozen stdout and repeat three times.

A fast execution of these nine tasks measures startup, dependency initialization, pool lifecycle and formatting. It cannot establish steady queue/lock superiority. More importantly, the ledger lock serializes its state transitions: parallel workers alone do not imply that workload has scalable parallel work. Use the frozen fixture unchanged as the semantic gate; build separate scalable, same-source Java/generated-Go benchmark variants only after the gate passes. Do not weaken ownership/cancellation assertions or remove the failure because it makes Go faster.

## New designs

`CountDownLatch` uses a mutex-protected count plus one closed channel. Every successful decrement participates in the mutex order; the final decrement closes the channel, publishing preceding arrivals to waiters. `AwaitTimed` validates `TimeUnit` before observing release, avoids timer allocation if already released, and allocates a timer only when it must wait. Keep that ordering: even a released latch must not bypass required argument validation. Saturated CountDown/GetCount still use the mutex; optimizing this is lower priority than queue allocation and global monitor lookup. An unconditional atomic decrement is wrong because the count must never become negative. Any CAS redesign must retain all-arrival publication, one close, zero-count construction, concurrent decrement/getCount and timeout behavior.

`Execution` now owns a `*Thread`. `ThreadCurrentThread` reads that field directly and falls back to the main-thread object for entry executions. Started threads install their own object; successful pool tasks share one named worker object and one execution token. Names are built once, and there is no global token-to-thread retention registry. This is a suitable low-overhead design: preserve it instead of adding goroutine-ID parsing or a locked global map. Lookup is constant work, but no JVM-relative advantage has been measured.

At initial inspection, uncaught `Execute` recovery installed `NewThread(nil)` into a new token while the original thread's `done` remained tied to the worker-loop defer. The replacement was not marked started and its completion channel was not closed on loop termination. This lifecycle issue was reported to the runtime owner; it may be fixed after this snapshot. Before timing failure recovery, verify old worker termination, replacement liveness/identity, same-pool naming policy, and replacement termination. `Submit` exceptions should remain Future failures and should not manufacture replacement identities. Exported Go entrypoints using the fallback main identity are an interop policy, not proof that arbitrary concurrent external Go callers have faithful Java thread identity.

## Concrete implementation proposals and acceptance gates

### 1. Clear removed List storage (lowest risk)

Proposed body of `List.RemoveAt`, for the runtime owner to apply:

```go
old := l.elements[index] // preserve existing bounds/exception evaluation first
last := len(l.elements) - 1
copy(l.elements[index:], l.elements[index+1:])
var zero T
l.elements[last] = zero
l.elements = l.elements[:last]
return old
```

The cleared slot is outside the resulting Java list, so Go's zero value is appropriate even where a visible Java String null uses a sentinel. Preserve order, returned object identity, and current bounds behavior. No change to capacity is needed. Validate tail/middle/head removal and a reference-containing element; do not retain a removed return value in the heap diagnostic. Acceptance: `retention_probe -mode list-remove` changes stale refs from 32 to 0 and releases essentially all 4MiB payload storage while the empty list itself remains live. An allocation-noise delta is acceptable; no numeric speedup is required to accept this lifetime repair. Then observe long-lived service live-heap plateau across repeated insert/remove batches.

### 2. Successful class initialization atomic fast path

Use a separate `atomic.Bool` successful flag; keep the existing state machine under its mutex. Maintain this order:

1. Preserve execution/receiver validation before any fast return.
2. If successful.Load() is true, return without locking.
3. Otherwise run the existing slow path, including recursion and erroneous state.
4. Only the execution that completed the entire initializer body and required dependencies successfully may publish successful.Store(true), after all initializer writes. Publishing while holding the existing mutex is easy to audit.
5. Recursive same-execution entry and failure must never publish success. Retain owner cleanup, condition broadcast, cause handling and zero-value coordinator support.

Proof requirement: an observing atomic load establishes publication of every initializer write before that successful store; all other state/cause access remains protected by the existing mutex. Do not read the ordinary `state` without locking, mix atomic/non-atomic accesses to it, or move initialization across Java active-use evaluation boundaries. Readers that see false continue to lock/wait and must still be released by the normal broadcast. Fast return before the initializing execution releases the state mutex is fine only because body/dependency initialization is already complete and no fast reader inspects protected mutable state.

Acceptance: existing recursive/order/error JVM oracles unchanged; concurrent readers observe initialized ordinary fields with `go test -race` clean; zero-value and nil validation unchanged. A focused ready-state allocation diagnostic remains zero. Profile confirms initialized `Ensure` no longer attributes mutex wait/acquisition. Scheduled generated-code trials should show consistent throughput/latency improvement beyond noise at 1/2/4 workers; report lack of improvement honestly if active uses are not hot in the app. Do not claim eliminating runtime lock overhead proves beating warmed JVM initialization checks.

### 3. Reuse executor queue capacity

Current pop clears its slot (good) then reslices `queue=queue[1:]`. In a drained one-task-at-a-time queue, each next append starts at zero capacity and allocates another backing array. Keep the original slice plus a head index, or use a circular buffer. Under the existing mutex, zero each popped closure; when fully drained reset head and length so storage can be reused. Compact/grow only when amortized work justifies it. A burst-capacity release policy may be added separately after retention measurements; avoid a guessed large permanent reserve.

Acceptance: deterministic low-cost sample below drops by about one allocation per sequential Execute task (observed total is two today; verify with allocation profile rather than assuming both sources). FIFO dequeue is preserved, though multiworker completion remains unordered. Producers never block merely because the queue has a bounded capacity. Shutdown rejects new work atomically, drains accepted work, and all Future/cancel/failure/termination oracles pass. Cleared closures cannot retain task payloads after completion. A saturated burst must not show worse asymptotic allocation or unbounded compaction cost. No batching dequeue without modeling worker failure/queued cancellation semantics: reserving many tasks per worker changes observable task ownership and cancellation opportunities.

### 4. Canonical object-owned monitors before lock micro-tuning

Ledger methods repeatedly access the same global monitor registry. A generated-object monitor attached to shared `ObjectInfo` could avoid that registry lock and permanent roots while preserving all hierarchy views. Keep the global fallback until array/runtime/external-object identity is solved; do not pretend sharding alone fixes retention. Acceptance: same-object exclusion, reentrancy, inherited aliases, wait/reacquire, notify ownership, panic unwinding and legacy calls pass; independent-request heap becomes bounded by live requests; profiles show unrelated-lock registry contention removed. Only then examine per-monitor lock transitions/guard escape. Do not remove Java synchronization from Ledger to manufacture a throughput win.

## Low-cost diagnostics from this round

`go run ./performance/runtime/cmd/lifecycleprobe` runs only 33 samples per allocation metric, no elapsed-time measurement. Go 1.27.1 Darwin arm64:

| Check | Observed |
|---|---:|
| Already-open timed latch wait | 0 allocations/call |
| Saturated CountDown + GetCount | 0 allocations/call |
| Sequential Execute on one persistent worker | 2 allocations/task |
| Worker identity stable across calls | true |
| Worker alive before shutdown | true |
| Worker stopped after termination | true |

Allocation counts include all work during the call/handshake and are diagnostic totals, not isolated proof of allocation sites. Repeated retention diagnostics reproduced the first-round deltas (baseline 16B, monitors 4,204,072B, removed list 4,195,352B with 32 stale refs, cleared list 40B). The map retry diagnostic still reports two equality callbacks after a disjoint-key insert. Raw files: `.campaign/performance/runtime-lifecycle-round02.json`, `runtime-retention-round02.jsonl`, `runtime-chmretry-round02.json`.

## Scheduled benchmark variants and decision rules

Do not edit frozen Sol fixtures. A dedicated benchmark source set should reuse equivalent task/ledger work, transpile all measured Go from that Java, and publish enough checksum/state to prevent elimination. Root must schedule a clear CPU window first. Keep the original three-seed fixture as a prerequisite, including queued cancellation and cleanup. Use explicit JDK21 and the resource/warmup protocol in README.md.

| Variant | Exact change from semantic gate | Parity / acceptance | Candidate advantage to test (hypothesis) |
|---|---|---|---|
| Persistent queue | One pool per trial, P=1/3/4, 1k/100k tasks; fixed seeded payload; queue each batch after previous batch completes | Same result checksum and state/count invariants, same worker cardinality; compare lifecycle separately | Reused buffer and goroutine scheduling may lower steady allocation/task and submit-to-complete latency |
| Shared vs independent ledgers | Separate variants: all tasks share one ledger, or each lane owns its own ledger; same variant on both runtimes | Identical transitions and totals; no removing synchronized modifiers | Object-owned monitor lookup may improve independent-lock scaling; shared lock measures unavoidable serialized work |
| Request-lifetime heap | Repeated bounded batches, fresh ledger/payload graphs per batch, drop completed batch handles | All results checked before dropping, fixed in-flight bound, post-GC heap/peak RSS reported | Removing monitor roots permits Go heap plateau rather than growth with completed requests |
| Parked-worker capacity | Same fixed pool count 4/64/256 in Java and Go, all workers reach entered latch then block on release | All entered, no task run on submitting thread, all release/finish/terminate; equal CPU budgets | Goroutine parking may reduce process memory relative to matching Java platform-thread pools; do not silently substitute Java virtual threads or change worker counts |
| Latch waves | P arrivals/waiters 1/3/16/64, distinct output slots then CountDown, all waiters verify all slots | No unpublished arrival, count saturates at zero, valid timeout/unit outcomes | Channel broadcast might be efficient at fanout; measure against warmed JVM, not assumed |
| Exceptional lifecycle | Deterministic queued cancel, callable throw, uncaught execute failure in separate batches | Callable retains worker; execute failure replaces identity and terminates old/replacement correctly; all Futures finish appropriately | Only after identity repair; optimize normal path without weakening exceptional behavior |

For each scheduled variant: exact outputs/checksums first; at least five independent runs, alternate runtime order, warm JVM in-process, fixed input/worker/CPU/memory envelopes. Keep construction/startup time separate. Track throughput plus p50/p95/p99 completion latency, allocated bytes/task, post-GC live heap, peak RSS, and thread/goroutine counts. A proposed speed optimization is accepted as a performance improvement only when the observed effect is repeatable beyond trial noise and semantic gates remain green. A measured improvement versus prior Go still does not establish a win over Java. Heap fixes may be accepted on bounded-lifetime evidence without a throughput improvement. Do not present forced-GC pause tests as normal service throughput.
