# Isolated hierarchy queue candidate

The candidate removes repeated queue allocations in `JavaTypeAssignable` and `registeredJavaViewCandidates` while retaining their locks, visited checks, and breadth-first order. On the unchanged generated hierarchy workload, `views(leaf, 64)` falls from **132 allocations to 2**. Source-level Java/Go parity, runtime suites, race suites, and new registration/overflow regressions pass. There is no measured throughput win or reduction in retained memory versus Java.

Only this performance directory and temporary copies were changed. No shared runtime/compiler/challenge edits, Git mutations, or new agents were used. The prior `object_identity` report remains unchanged.

## Concrete patch

[queue.patch](queue.patch) changes only two local queues in `stdjava/reference_arrays.go` at frozen revision `8cb6cd529c124802d9d004ef43d41dffe16f1085`:

- Start with eight `TypeID` slots in a local array.
- Read the next entry by an increasing head index instead of removing the first element of the slice.
- Append successors exactly as before, allowing ordinary slice growth beyond eight entries.

No cache, additional global state, receiver lookup, metadata mutation, synchronization removal, or ObjectInfo layout change is introduced. The zero-context patch was applied to a separate temporary baseline file and produced exactly the tested candidate bytes. If applying through Git, use `git apply --unidiff-zero queue.patch`; implementation approval remains with the runtime owner.

| Artifact | SHA-256 |
|---|---|
| Frozen archive | `886928a3a2846254e1f87373d93a02dbd171d12b2b1116c522314ee09a9ea2c2` |
| Baseline `reference_arrays.go` | `88f01dba105053ddae561a4141d695e85b3c7324ea3ca0952d96c779fddbf3a9` |
| Candidate `reference_arrays.go` | `66e84bf35a711e0793b9b2a897d813840d8b2df20d0be0781ca9aef9df60b09d` |
| Patch | `17308a763a802719561dfec04fe4ff3737f53805b5fbf65dc2415f6aa0d3870e` |
| Unchanged generated hierarchy application | `61b9f5202122db823186c22d53c9dcaa030680fa1a3e8a07c0d0a4f92e3c81fb` |
| New generated semantic oracle | `a2482fbefdad6e0ea763c3b74519350424a249f9c29092f82bdc4d520be00778` |

## Ordering and lifetime argument

At each iteration, the candidate's unprocessed suffix `queue[head:]` equals the baseline's remaining queue. Both inspect the same current node and execute identical early returns, empty-node checks, visited checks, and successor append order. Increasing the head and preserving the consumed prefix cannot change the order of unprocessed entries. Both traversals hold the same registry read lock for the entire walk. Registration replacement still acquires the original write lock; no cached answer survives replacement or late registration.

The eight-slot buffer and any overflow slices are local. Queue elements are type identifiers, not instance pointers. No new root keeps generated objects alive. For large/deep graphs the candidate keeps consumed entries in its temporary queue until return, so peak temporary capacity depends on total enqueued entries rather than the remaining frontier. This is a tradeoff to examine for unusually large graphs; no universal memory or stack-cost improvement is claimed. The visited map and the materialized fallback result slice remain unchanged.

## TDD and semantic validation

[regression_test.go.txt](harness/regression_test.go.txt) was installed only into the temporary baseline and candidate runtime copies. Before patching, semantic regressions passed and the allocation expectation failed: small leaf-to-base/interface/Object checks allocated **2 / 1 / 3** objects instead of zero. After patching, the same allocation regression passed at zero.

The new regression suite covers:

- Late ancestor registration and replacement of a previously resolved interface edge.
- Exact BFS result order with superclass/interfaces, duplicate interfaces, cycles, and missing targets.
- Forty interfaces and a forty-edge chain, beyond the eight-entry stack buffer.
- Concurrent registration replacement and readers; every observed traversal must represent one complete registration state.
- Reference-array covariance, primitive-array invariance, nested primitive arrays, Cloneable/Serializable, typed null, String null, valid base/derived and erased views, and ClassCastException.

Both frozen baseline and candidate passed the complete `stdjava` suite and complete `stdjava -race` suite. The intentionally failing allocation guard was excluded from the baseline full suite; allocation guards were excluded from both race suites, with exact counts tested separately without race instrumentation. All semantic regressions ran in those suites.

[QueueSemantics.java](source/QueueSemantics.java) adds a JVM oracle for overridden/interface dispatch through array-derived views, successful downcasts, null casts, bad casts, covariant array-store failure, and nested primitive arrays viewed as Object[]/Cloneable. JDK 21, baseline generated Go, and candidate generated Go all output **264**. The compiler is frozen and the generated bytes are identical between runtime variants.

The existing hierarchy source is reused byte-for-byte. Two seeds × three retention modes × Java/baseline/candidate produced **18 fresh processes** with exact stdout parity, including checks after GC that a retained base view still exposes the derived object. Each process performs two full warmup scenarios followed by the measured scenario. Both runtime copies receive the same read-only registry-count helper after their runtime suites; the generated application is never patched.

## Allocation evidence

Three fresh allocation-test processes for each variant produced the same results. Each operation uses `testing.AllocsPerRun(64, ...)`; fixed inputs are prepared outside the measured closure and return values escape to sinks. The generated entry points reuse an explicit `Execution`. Counts describe this Go build, not JVM allocation counts.

| Operation | Baseline | Candidate |
|---|---:|---:|
| Construct three-level graph and 8-byte payload | 7 | 7 |
| Bare ObjectInfo / generated ObjectInfo | 1 / 2 | 1 / 2 |
| Canonical equality / exact leaf / base-to-leaf | 0 / 0 / 0 | 0 / 0 / 0 |
| Leaf-to-base | 2 | 0 |
| Leaf-to-interface | 1 | 0 |
| Erased Object descriptor to concrete base view | 8 | 2 |
| Warm getClass | 0 | 0 |
| Generated `views(leaf, 1)` | 6 | 2 |
| Generated `views(leaf, 64)` | 132 | 2 |

The generated loop's allocation growth is eliminated for this hierarchy. The remaining two allocations in erased-view recovery come from unchanged traversal/result work; this candidate does not claim all hierarchy shapes are allocation-free. A throughput benefit from lower GC work is a hypothesis requiring a separately scheduled, matched JVM/Go benchmark.

## Retained-heap evidence and limits

The workflow allocates three measured batches of 32 objects, each with a 131,072-byte payload: **4,194,304 payload bytes per batch**, **12,582,912 bytes across the measured scenario**. Modes drop all source roots, retain one base view, or retain the current batch. Heap observations are taken after two explicit GCs; all numbers below are decimal bytes relative to the scenario's empty baseline.

| First measured batch, range over seeds | JVM | Baseline Go | Candidate Go |
|---|---:|---:|---:|
| Drop all | 0 | 16 | 16–64 |
| Keep one base view | 131,112 | 131,264–131,312 | 131,264 |
| Keep current batch | 4,195,728 | 4,200,512 | 4,200,512–4,200,560 |

Across all measured batches, candidate deltas were bounded at 96 bytes with no source roots, 131,264 bytes with one root, and 4,205,928 bytes with a current batch. After clear they ranged from 0 to 5,416 bytes; baseline clear deltas ranged from 0 to 128 bytes. One candidate process had a roughly 5 KiB residual; this run does not diagnose it or present it as object-retention improvement. The bounded observations show no accumulation proportional to prior 4 MiB payload batches.

Every Go heap phase had exactly **99 type entries, 1 class literal, 5 class descriptors, and 0 monitor records**. Allocation probes likewise preserved all registry counts except the expected first getClass literal. Later JVM snapshots have the same 36,104–168,104-byte residual behavior documented in the original study. Those values are preserved in the evidence and prevent exact retained-size or memory-superiority claims. Actual monitor-acquisition lifetime is outside this zero-monitor workload.

## Reproduction and acceptance

Run `python3 performance/runtime/object_identity_candidate/run.py` after coordinating allocation/heap measurements. It archives the frozen revision read-only, performs the baseline red/candidate green test, runs runtime/race suites, generates both source workloads, compares JVM output, and records three allocation repetitions per Go variant. It does not modify live production code.

Environment: Darwin arm64; Go 1.27.1; explicit JDK `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`, Java 21.0.6+8. JVM uses Serial GC, two active processors, `-Xms32m -Xmx256m`; Go uses `GOMAXPROCS=2`, `GOMEMLIMIT=256MiB`, `GOGC=100`. Heap caps are comparable configured limits, not identical GC policies. Application/measurement subprocesses have a 30-second cap; test/build commands have separate bounded timeouts. Test-tool elapsed times are incidental and are not used as performance results.

Raw run: `.campaign/performance/object-identity-candidate-20260928T001930`. [evidence.json](evidence.json) tracks exact observations and hashes; raw commands, test output, generated sources, and full runtime fingerprints remain in that directory. The exact tested regression source is preserved there; the owned copy was subsequently gofmt-formatted without changing logic.

This candidate meets the requested isolated acceptance checks. Before a production change, apply it to the current runtime with the same regressions and rerun the current branch's runtime/reference-array/generated-source checks, including recent monitor alias/constructor-publication tests. The frozen experiment predates that repair. Benchmark throughput only in a scheduled comparison with matched work, cores and heap configuration, separating startup and warmed steady state.
