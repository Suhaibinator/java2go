# Accepted round01 data application: allocation audit

The clearest small general candidate is **allocation-free UTF-16 comparison, starting with a guarded ASCII path**. A second candidate is **ASCII substring extraction without decoding the entire source to runes**. Both affect unchanged Commons Codec/application code through standard String operations. They do not replace the dependency with fixture-specific code. No candidate was applied, no challenge/generated implementation was edited, and no JVM/Go speed claim is made.

## Provenance and validity

Accepted run: `.campaign/runs/20260928T050238Z-2636650079`, with `passed: true`. Its `race_enabled: true` and approximately one-second Go process observations make that correctness report unsuitable for performance comparison: race instrumentation and shutdown behavior are present. A new isolated non-race build used the **unchanged generated Go files** from that run and runtime commit `563fddbe837ba558ae3d1f57c9a947a179f845af`; every original generated `.go` hash was checked after copying. Only a separate instrumentation main and a module replacement were added in scratch.

The profiler invokes the original generated main twice in one process: one warmup followed by one observed call. It uses `MemProfileRate=1`, snapshots allocation stacks around the observed call, and forces GCs to flush profile records. These settings deliberately invalidate wall-clock comparison. `runtime.MemStats` deltas measure allocations inside the original main; file hashing and profiler snapshots are outside those deltas. Profiling includes required file/resource behavior and normal stdout formatting. Output path lengths were held equal across the final three-trial series; an earlier exploration used shorter paths and is not mixed into the table.

For each of seeds 17/41/97, a fresh live JDK21 oracle matched the accepted stdout and all three file hashes. Every warm and observed Go invocation matched that live stdout and all file hashes, including the warm invocation's files before they could be overwritten. Final series: **18 verified Go application invocations and 3 live Java invocations**, in addition to the initial exploratory checks.

| Seed | Allocated bytes per observed main, 3 fresh trials | Mallocs per observed main, 3 trials |
|---|---|---|
| 17 | 289136, 289088, 289088 | 4410, 4409, 4409 |
| 41 | 281256, 281256, 281256 | 4240, 4240, 4240 |
| 97 | 263912, 263912, 263912 | 3992, 3992, 3992 |

The small 48-byte/one-malloc variation for seed 17 was retained. Other-process CPU contention does not directly enter these per-process counters; runtime bookkeeping, pools, paths and profiling remain relevant controls. These are Go allocation counts only, not Java allocation comparisons or heap/RSS comparisons.

## Attribution and candidate ranking

Selected inclusive stack totals from seed 17, first final-series trial:

| Stack contains | Attributed bytes | Profile objects | Interpretation |
|---|---:|---:|---|
| StringCompareTo | 11312 | 167 | UTF-16 rune buffers during account/reject sorting and TreeMap grouping |
| StringSubstringRange | 18824 | 138 | Full-string rune decode for short digest-prefix extraction |
| StringFromChars | 24368 | 158 | Temporary rune buffer plus final string for generated Hex output |
| javaInputStreamReader | 32960 | 12 | Java/Go reader bridge and nested read-copy allocations |
| InputStreamReadIntoExecution | 16384 | 4 | Nested signed/unsigned byte-buffer copy; included in reader row |
| StreamGroupingByDownstreamWith | 6496 | 211 | Group materialization plus downstream map/comparison work |
| StreamSortedWith | 10512 | 260 | Sort copies plus comparator allocations; overlaps StringCompareTo |
| NewStream | 344 | 10 | Direct stream slice-copy sites, including grouped substreams |
| reflect. | 2088 | 46 | Observed reflection allocations; not proof reflection dominates CPU |

Rows **overlap** and must not be summed. Profile objects and MemStats Mallocs count tiny allocations differently; use the top-level MemStats deltas for total malloc count. Profiles identify allocation sites, not CPU time.

1. **String comparison without materialized UTF-16 buffers.** `stdjava.StringCompareTo` currently obtains two `StringChars` slices before examining the first unequal unit. Reject locations and account IDs are ASCII; grouping labels include NFC accents, Japanese and emoji. A general ASCII guard can compare bytes and return the exact Java difference for ASCII pairs without allocating. A complete implementation can iterate UTF-16 units from UTF-8 incrementally, emitting high/low surrogates in order. Preserve exact difference (not merely sign), prefix length difference in UTF-16 units, null-check timing and existing malformed/unrepresentable-string boundaries. Test ASCII, BMP, supplementary-versus-BMP ordering, unequal surrogate-pair halves where supported, equal prefixes, empty strings, and comparator ties/stability. Do not use Go bytewise comparison for all Unicode: UTF-8 order differs from UTF-16 for supplementary versus some BMP values.
2. **Guarded ASCII substring.** Accepted parsing calls `sha256Hex(...).substring(0,12)` repeatedly. The seed 17 parse path alone allocates54 full 64-rune temporary slices (13,824B), then short result strings. A generic runtime ASCII guard permits byte-index extraction with the same bounds/null behavior; copy the requested range when needed to avoid retaining arbitrarily large backing strings. No Codec-name recognition or digest-specific lowering is required. Unicode fallback must not silently turn Java UTF-16 indices into rune indices; the existing general UTF-16 storage limitations remain a separate correctness prerequisite for broader optimization.
3. **Direct UTF-16-to-UTF-8 construction.** Generated `Hex.encodeHexString` correctly traverses real dependency code into a char array, then runtime `StringFromChars` builds another rune slice before building the final Go string. Stream validated UTF-16 units into a byte builder or a single measured output allocation, preserving complete surrogate pairs, existing isolated-surrogate failure behavior and independence from future source-array mutation. The profile's StringFromChars row includes necessary final string allocations, so the whole 24,368 B is not a removable budget.
4. **Typed I/O buffer bridging, with override preservation.** The adapter allocates a signed buffer, calls the generated tracked Java read override, delegates to a native read that allocates an unsigned buffer, then copies back through both representations. Seed17 allocates four 4 KiB buffers in each of those layers, in addition to decoder/bufio buffers. This is large, but a higher-risk candidate: retain override calls, byte counts, read/EOF/error ordering, close suppression and Java array mutation behavior. Do not bypass `TrackedInputStream` or globally reuse a buffer whose identity/contents a subclass could retain. A proven nonescaping/internal-buffer bridge or safe owned representation is needed before changing this.
5. **Stream copy/collector lowering after the String work.** Generated accounts path is MapValuesView.Slice→StreamOfSlice(copy)→StreamSortedWith(copy)→ToList(copy). Grouping materializes Lists, copies group slices into Streams, maps balances into primitive slices and reduces. Fresh ephemeral ownership could remove redundant copies; a summing collector can accumulate primitive int values directly. Preserve source encounter order, stable sorting, mapper/classifier invocation order, TreeMap comparator-equivalent keys, null-key failure, integer overflow and final boxing. In this small accepted application direct NewStream copies account for only 344 B, and much of the inclusive stream cost is actually comparator decoding. Do not prioritize copy removal over the measured larger string costs without a larger correctness-matched workload.

## What is not established

- Codec calls in this application are mostly concrete execution-companion calls. The tracked InputStream uses deliberate typed interface bridges. A generic claim that dynamic dispatch or reflection dominates is unsupported by these allocation profiles.
- Grouping lowers balances as primitive int values and boxes the downstream result once per group. Boxed values outside Java's cache still matter semantically; do not delete boxing or alter identity. Absence of a BoxInteger frame in an inlined/tiny-allocation profile is not proof of zero boxing.
- `Hex.ToDigit` calls its class-initialization guard per decoded character; the initialized guard takes a mutex. This is a plausible CPU candidate already covered by the separate runtime audit, but allocation data does not quantify its cost. Use a separately scheduled CPU profile and class-initialization semantic tests before changing it.
- Sorting/grouping strings have accepted Unicode output parity here; that does not prove every StringBuilder/String substring edge case in the runtime. The benchmark must retain the accepted accented/Japanese/emoji/malformed cases rather than replacing them with ASCII-only input.

## Reproduce

```sh
python3 performance/codegen/round01_data/prepare_allocations.py /tmp/data-allocation-check
python3 performance/codegen/round01_data/run_allocations.py /tmp/data-allocation-check --trials 3
```

Use an empty scratch directory. This performs allocation diagnostics and live semantic checks only, never a sustained timing run. `allocation_evidence.json` retains the compact final-series evidence; full allocation stacks, outputs, frozen generated code and metadata from this investigation are in `/tmp/java2go-round01-data-allocations/`. `BENCHMARK_PLAN.md` specifies the separate sustained comparison and its quiet-window gate.
