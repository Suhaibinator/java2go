# Isolated slot-clearing candidate — validated, not applied

The reviewable change is [list-remove-clear.patch](list-remove-clear.patch). It modifies only the body of `List.RemoveAt` in an **isolated runtime copy**, preserving the new fixed-size-list guard. Live production/compiler/challenge files were not edited. This is a measured retained-heap improvement over the current Go runtime, not a speed win over Java and not a campaign gate.

The existing [ListRetention.java](source/ListRetention.java) source was unchanged (SHA256 `71f10421f36639ec6fc4d59541222a4dfd0035a426deba734f88c0bd498dcbe3`). It was transpiled once, then the exact generated application source was built against baseline and candidate runtime snapshots. Java used the same application source. Each process kept one list alive through four fill/remove cycles; three fresh process triplets ran at each size. Semantic stdout matched Java/baseline/candidate in every triplet.

| Live empty list after removal | Java ArrayList | Baseline generated Go | Candidate generated Go |
|---|---:|---:|---:|
| Previous payload batch: 8 MiB | 56 B | 8,392,208 B | 528 B |
| Previous payload batch: 16 MiB | 0 B | 16,780,816 B | 528 B |

Values are median final post-remove heap minus that same process's later post-clear heap. All three samples in each cell were identical. The candidate retains only the small reusable list backing storage/instrumentation, not the removed array payloads or wrappers. The residual 528 B does not imply Java uses zero list capacity: Java's `clear` retains capacity while this Go runtime's `Clear` drops it. Within-runtime controls establish payload release; these numbers are not whole-process memory comparisons.

No wallclock performance claim is made under concurrent campaign load. Java SerialGC/full-GC instrumentation and Go's collector/accounting differ. Nominal CPU/memory limits and the source/workload were matched as in README.md. The performance-agent window was coordinated and released after these short probes.

## Proof obligations and observed checks

The patch keeps `requireResizable()` first and the existing indexed read second. Thus fixed-size removal continues to reject before bounds access; ordinary invalid-index evaluation stays where it was. `old` preserves the returned reference. `copy` uses the same source/destination range as the old overlapping append; survivor order and capacity remain unchanged. The sole new observable storage mutation is zeroing the no-longer-visible last slot before shortening the list.

- The complete copied `stdjava` test suite passes on baseline and candidate.
- The additional deterministic slot test fails baseline and passes candidate. It checks head/middle/tail removals, returned pointer identity, surviving element order, unchanged capacity, and complete zeroing of a struct containing a pointer, slice and string. Repeated tail removal leaves no stale pointer slots.
- The new [JVM semantic oracle](semantics/ListSemantics.java) matches both baseline and candidate. It covers `Set` returning the old null value; head/tail and object-overload removal; duplicate-first-match removal; null/absent-null removal; reference identity; empty/negative/high bounds failures; enhanced-for order; observing nonstructural `Set` during iteration; `Arrays.asList` write-through; fixed-size indexed/object removal failures including invalid index; missing-object removal returning false; and backing-array updates observed during fixed-list iteration.
- Exact semantic output: `true:true:true:true:true:true:true:true:ENH:true:Nba:true:xza:true:RIO:true:AA`.

These checks establish supported iteration behavior, not the full Java Iterator contract. Current `CollectionIterationElements` ranges the ordinary backing slice and lacks Java iterator structural-modification tracking. Removing structurally during that traversal can expose stale or now-zeroed trailing slots instead of Java's fail-fast behavior. This is a pre-existing unsupported parity area; the patch must not be marketed as fixing it. An external Go caller holding a resliced backing array across mutations can also observe the new zero, which is the intended lifetime change rather than a Java-visible element change. Do not preserve unreachable references merely to maintain that diagnostic/external storage observation.

The artifact uses zero-context diff format; apply it with `git apply --unidiff-zero` only when this candidate is scheduled for implementation.

Recommended acceptance: review/apply this small patch through the runtime owner, rerun the existing runtime and targeted generated parity suites, then rerun this candidate's unchanged source/heap and slot checks. Keep all fixed-size/null/exception tests. Payload growth should disappear at both sizes; no speed threshold is needed for this memory-lifetime repair. Broader collection/iterator correctness should remain a separate explicit issue.

## Reproduce / inspect

```sh
python3 performance/runtime/list_retention/validate_candidate.py
```

The script copies only runtime sources/module files to temporary baseline/candidate modules, applies the single replacement only there, runs runtime/slot/oracle checks, builds shared generated applications against each snapshot, and saves raw evidence. It creates/refreshes the reviewable patch file but never applies it to the live repository. It deliberately asserts a failing baseline slot test and aborts on unexpected outcomes or semantic mismatches.

Evidence: `.campaign/performance/list-candidate-20260927T221405/` contains `results.json`, commands, test output, JVM/Go semantic output, generated application sources, patch and all per-process memory samples. Isolated modules from this run: `/var/folders/c1/bq3rcd9d3y54bw8n_fp4x0sr0000gn/T/java2go-list-candidate-dcg6whlc/{baseline,candidate}`. The live list SHA256 is recorded in results and checked unchanged across the run. Reproduction snapshots the then-current runtime; if its removal body has changed, the script stops for review rather than guessing another patch.

## Follow-on IO/decoder/resource audit (source evidence only)

Read-only follow-on inspection found these general candidates; none was modified, timed, or presented as a Java performance win:

1. **Avoid class-resource byte conversion round trips.** `class_resources.go:GetResourceAsStream` calls `fs.ReadFile`, then `signedByteArray`, then `NewByteArrayInputStream`, whose `javaBytes` converts signed elements back to unsigned bytes. This copies the full resource at least twice after `ReadFile` and creates an intermediate Java array wrapper. For immutable registered roots with an independently owned read result, a private constructor taking the already-read `[]byte` can avoid those conversion copies while still returning an independent stream/cursor. Do not replace public ByteArrayInputStream array semantics without checking Java's shared backing-array behavior. Validate repeated resource opens, independent positions, missing/relative resource names, exact decoded bytes, custom source lifecycle, and large-resource allocated bytes. Snapshot the registered roots under its RLock and release it before filesystem reads; the current read lock spans the whole IO/conversion operation. Preserve root search order and concurrent registration semantics.

2. **Release closed buffered-stream storage.** `BufferedInputStream.Close` marks closed and closes the source but retains `reader`, `source`, and `execution`; `BufferedReader.Close` only invokes the captured close function and retains its buffer/source chain. If a service keeps closed wrappers, their buffers and underlying byte-array payloads remain reachable. Installed JDK21 source explicitly clears BufferedInputStream's buffer/input and BufferedReader's input/char buffer on close. A candidate should clear internal references after taking the source needed for close, with the intended error/closed state established even if source.close throws. First define synchronization, reentrant/double-close and post-close-read behavior—do not nil a field that the current read path dereferences without a closed check. InputStreamReader's JDK StreamDecoder itself retains its input in the inspected implementation, so do not generalize every closed-reader reference into a Java-relative leak. Also keep ByteArrayInputStream and ByteArrayOutputStream close no-op semantics; their bytes remain usable after close.

3. **Audit per-read bridge scratch allocation.** `InputStreamReadIntoExecution` allocates a length-sized `[]byte` and copies it into Java `[]int8`; `FileChannel.Read` similarly allocates a remaining-sized temporary. These are allocation candidates for repeated reads into a caller-reused Java buffer. Prefer an explicit compatible backing representation or scoped reusable scratch, subject to synchronous consumption, reentrancy, concurrency, partial-read+error and bounds/override-order proofs. A cache must not retain arbitrary maximum request sizes forever. Custom InputStream overrides and current execution propagation cannot be bypassed to remove copies.

4. **Keep decoder state scoped to one stream.** The UTF8/UTF16 transformers process chunks directly and keep only small decoding state; incomplete bytes are held by transform.Reader. This is substantially different from whole-file eager decoding and no payload-proportional decoder leak was established. The current creation allocates a transformer/transform.Reader per reader; pooling may be less valuable than the proven whole-buffer copies above. Any reuse must reset BOM/endian/incomplete-unit/error state and release the old source/execution. Malformed-unit grouping, short-source/short-destination handling, chunk-boundary behavior and exception timing are semantic constraints, not optional fast-path costs. Do not skip these transformations merely because ASCII is common.

Prioritize the validated list fix first, then resource conversion and closed buffered-wrapper lifetime probes with matching JVM sources. Quantify their allocation/retention before ranking a throughput optimization.
