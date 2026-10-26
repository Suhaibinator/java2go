# Canonical source-object identity: lifetime and allocation

The frozen runtime does **not** globally retain each generated `ObjectInfo`. Cyclic source hierarchies remain collectible when the application drops their references and never acquires an actual monitor. Keeping a superclass view intentionally keeps its most-derived object alive and permits a cast back after GC. Repeated nominal view checks do allocate; that is the stronger optimization target than weakening canonical identity.

This is an allocation/reachability audit, not a speed comparison or campaign gate. Production/compiler/challenge files were not changed. The separate live monitor alias/null/noninsertion repair is outside this frozen experiment. Actual monitor acquisition still has the independently reported global-root lifetime problem.

## Frozen inputs and reproduction

Run `python3 performance/runtime/object_identity/run.py` from this worktree after coordinating the measurement window. The runner uses read-only `git archive` of `8cb6cd529c124802d9d004ef43d41dffe16f1085`, builds that compiler, strictly transpiles the shared Java source, and builds two binaries with the **same generated application bytes**. The instrumented runtime differs only by a read-only registry-count helper; it is not an optimization candidate. No generated application edits occur.

Environment: Darwin arm64, Go 1.27.1; explicit `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`, Java 21.0.6+8. JVM: Serial GC, two active processors, 32 MiB initial / 256 MiB maximum heap. Go: `GOMAXPROCS=2`, `GOMEMLIMIT=256MiB`, `GOGC=100`. These are comparable configured bounds, not identical collector budgets. Every measurement process is fresh and capped at 30 seconds. No elapsed-time comparisons are used.

Final raw run: `.campaign/performance/object-identity-20260927T235324`. Commands, raw stdout/stderr, generated source, archive, full file hashes, escape analysis, and results are preserved there. Small tracked [evidence.json](evidence.json) includes all process observations, output hashes, environment, and runtime manifest hashes.

| Input | SHA-256 |
|---|---|
| Frozen archive | `886928a3a2846254e1f87373d93a02dbd171d12b2b1116c522314ee09a9ea2c2` |
| Shared Java source | `7efbc20e1fc3325c11ae7b582393fbb499160baa0ad520ca500bd0779f356e80` |
| Generated application | `61b9f5202122db823186c22d53c9dcaa030680fa1a3e8a07c0d0a4f92e3c81fb` |
| Baseline runtime manifest | `d55a9f2c1e9ee8c6eae3b099cbd283b62cc5e1553aad015b3ee6f18ef46e001f` |
| Instrumented runtime manifest | `f6ba8204596371c58e8564670eb804969385d46354a199f39496a218e1ae0757` |

## Source-level reachability evidence

The shared source constructs `GraphLeaf -> GraphMiddle -> GraphRoot`, gives each leaf a 131,072-byte payload and a self-reference through its base type, and exercises reference arrays, downcasts, overridden dispatch, interface calls, equality, and dynamic `getClass`. Each batch has 32 objects: 4,194,304 payload bytes. Three batches run in each scenario; two whole warmup scenarios precede the measured one in the same process. Seeds 17 and 97 alter payload/checksum values.

Three modes drop all references, retain only the last base view, or retain the current batch of base views. Retention checks run **after** the GC snapshot, proving the base view still resolves to a usable most-derived leaf; final clear checks return zero. Exact stdout agrees for Java, frozen Go, and instrumented Go in all six seed/mode combinations (18 fresh processes). This proves parity of the exercised semantics, not unsupported Java features in general.

Post-GC heap deltas are decimal bytes relative to the scenario's empty baseline. The table uses the first measured batch, where JVM observations are consistent across both seeds:

| Mode | Java, both seeds | Frozen Go, range across seeds | Instrumented Go, range |
|---|---:|---:|---:|
| Drop all source graphs | 0 | 16 | 16–64 |
| Keep one base view | 131,112 | 131,312 | 131,264–131,312 |
| Keep current 32-object batch | 4,195,728 | 4,200,512 | 4,200,512 |

Across all three measured batches, frozen Go retained at most 64 bytes with no application roots, 131,440 bytes for one root, and 4,200,560 bytes for a current batch. Clearing roots reduced frozen Go deltas to 0–176 bytes. The instrumented process has similarly bounded behavior (0–144 bytes after clear). There is no growth proportional to the 12,582,912 payload bytes constructed per measured scenario.

Later JVM snapshots contain additional 36,104–168,104-byte post-clear residuals, and the single-root case does not consistently fall immediately on clear. Heap snapshots do not establish exact object death or explain those residuals; startup/JIT/stack-liveness effects remain unseparated. They do not support a claim that Go retains less than Java. The initial diagnostic with less warmup is retained at `object-identity-20260927T235136`; the failed intermediate driver build is retained at `object-identity-20260927T235258`. Only the final run supplies the table above.

Instrumented runtime counters are exactly **99 type entries, 1 class literal, 5 class descriptors, and 0 monitor records** at every heap phase. The allocation probes begin with zero class literals and end with one, due to the first `getClass` for `GraphLeaf`; all other counters stay unchanged.

## Why the graph is collectible

In frozen `stdjava/reference_arrays.go:290–352`, `ObjectInfo` contains the dynamic type and a bound view function. `NewGeneratedObjectInfo` captures the most-derived receiver. The generated hierarchy root owns one shared identity; middle and leaf promote that same pointer. Generated view methods return existing subobject pointers and do not create view wrappers.

The resulting cycle is local: leaf -> middle -> root -> ObjectInfo -> bound receiver -> leaf. The root's virtual-dispatch receiver and source `peer` add further local cycles. None is a global root. Retaining a base pointer must retain the derived object because subsequent casts, dynamic dispatch, and fields remain observable.

The global type registry contains type identifiers and hierarchy edges; `classLiterals` contains one `Class` per requested type; the reflection registry contains class descriptors and metadata. In generated source, reflection initialization/construction callbacks refer to class-level functions and do not capture instances. Custom runtime callers could register their own capturing closures, but this probe does not do that. The audit does not establish unloadability of dynamically registered types or class loaders.

Canonical identity equality in `stdjava/reference_identity.go` compares the shared ObjectInfo pointers. It neither registers an object nor replaces its identity. An acquired monitor is a different lifetime path: a global registry/anchor can turn the otherwise local cycle into a process root. This experiment intentionally uses zero monitors; removing synchronization is not a proposed optimization.

## Repeated allocations and ranked recommendations

Three fresh `testing.AllocsPerRun(64, ...)` processes give identical allocation counts. Inputs are prebuilt where appropriate, returned values escape into global sinks, and generated methods share one `Execution` so exported-call token construction does not contaminate the counts. These are Go allocation counts, **not** Java allocation counts or timing results.

| Operation | Allocations/call |
|---|---:|
| Construct three-level graph with 8-byte payload | 7 |
| `NewObjectInfo` without provider | 1 |
| `NewGeneratedObjectInfo` for existing leaf | 2 |
| Canonical equality | 0 |
| Exact leaf view / base-to-leaf view | 0 / 0 |
| Leaf-to-base view / leaf-to-interface view | 2 / 1 |
| Erased `Object` descriptor requesting concrete base view | 8 |
| Warm `ObjectGetClass` | 0 |
| Generated source `views(leaf, 1)` / `views(leaf, 64)` | 6 / 132 |

1. **Reduce temporary allocation in nominal hierarchy traversal.** `JavaTypeAssignable` front-slices its queue then appends successors, exhausting usable capacity on simple chains. `registeredJavaViewCandidates` repeats that walk and materializes a result slice for erased views. The measured view counts and generated loop's additional 126 allocations for 63 additional iterations make this the most direct target. First candidate: a head-index traversal over a small stack-backed queue with overflow, retaining the visited check and current BFS order. A second candidate could reuse immutable per-type ancestry metadata. Expected benefit is a hypothesis: fewer short-lived allocations and less GC work on array/cast-heavy applications; no speedup is measured here. Preserve nominal checks before structural Go assertions, array covariance, primitive distinctions, null/cast exceptions, cycles and duplicate interfaces, and erased-view selection order. Any cache must handle out-of-order registrations and `RegisterJavaType` replacement/invalidation under concurrency; a fixed process-lifetime cache keyed by arbitrary requests must not introduce unbounded retention. Acceptance: all current runtime/reference-array suites, a source JVM oracle covering casts/interfaces/generics/null/bad casts and registration mutation tests, unchanged output in this diagnostic, no additional global roots, and allocation reduction in both microprobes and generated `views`.
2. **Consider eliminating the separate bound-method closure allocation.** `NewGeneratedObjectInfo` costs two allocations versus one for bare `NewObjectInfo`; the implementation stores a method value capturing the receiver. A receiver interface stored in the same identity object could make the generated path one allocation, while preserving a compatible custom-provider path. This is a proposal, not an implemented or measured candidate. Do not create a global receiver table. Preserve identity publication before constructor callbacks, one identity across all subobjects, downcast recovery from a retained base, virtual dispatch, and immutability after construction. Acceptance: `NewGeneratedObjectInfo` 2 -> 1 and graph construction 7 -> 6, equivalent source output/GC behavior, constructor callback and cross-view monitor tests, and race checks. A larger interface-containing ObjectInfo may offset some bytes saved; measure bytes as well as allocation counts before choosing it.
3. **Treat flattening hierarchy allocations as a larger compiler project.** The generated graph allocates leaf, middle, and root separately, in addition to metadata and the array representation. Embedding storage could remove allocations but changes constructor installation, aliases, promoted method behavior, nil receivers, and interface dispatch. Benefit remains hypothetical and risk exceeds the two bounded candidates above. Require broad generated hierarchy/construction parity before any benchmark claim.

No optimization patch is included: this audit identifies collectible ownership and concrete allocation sites while the runtime owner repairs monitor correctness. The diagnostic's original exploratory source used a direct base `getClass` call that the frozen compiler did not lower correctly; that source/output remain in `object-identity-preparation/initial-*`. The final source expresses `getClass` through `Object` locals and uses the recovered leaf for its interface view. This limits the asserted parity to those supported paths without modifying the generated application.
