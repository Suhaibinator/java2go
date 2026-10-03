# Allocation-free String search experiment

The isolated candidate removes UTF-16 slice materialization from `String.indexOf` / `lastIndexOf` while preserving the checked Java results. **The actual generated delimiter call falls from 4 allocations to 1 (about 100 to 16 bytes per call). Direct helper calls fall to zero.** No wall-clock or faster-than-Java claim is made. No production source or generated acceptance output was modified.

The inspected generated call is `stdjava.StringIndexOf(stdjava.StringRequireNonNull(row), int32('|'))` in `.campaign/runs/20260928T053557Z-2909822102/generated/java2go_scc_63616d706169676e2f636f6e63757272656e74322f696f/63616d706169676e2f636f6e63757272656e74322f696f_PlanIO.go:42`, from `testfiles/campaign/round02/concurrent/src/main/java/campaign/concurrent2/io/PlanIO.java`. This fixture searches only five input rows; that workload alone cannot establish an important application win.

The current runtime converts the receiver and string needle into UTF-16 rune slices before searching. It also routes known strings through `StringRequireNonNull(any)`; compiler escape diagnostics and allocation probes implicate concrete-string boxing into that generic helper. The candidate uses a concrete-string sentinel check with the existing exception message. The generated outer generic null check remains, explaining the residual allocation.

## Candidate and semantic boundaries

`candidate.go.txt` is executable candidate source installed only into a runtime snapshot by `prepare.py`. It preserves the existing public signatures and receiver/from-index/needle validation order.

- ASCII integer needles use `strings.IndexByte` / `LastIndexByte`, converting the resulting byte prefix length back to UTF-16 units.
- Other integer needles stream over UTF-8 scalars while tracking UTF-16 positions. Supplementary needles match a complete scalar; surrogate-valued integer needles match either individual UTF-16 unit of a supplementary scalar.
- Valid UTF-8 string needles use `strings.Index` / `LastIndex`. Forward starts inside a surrogate pair round to the next scalar boundary; backward starts round to the preceding scalar boundary. The backward slice includes the complete needle at the final allowed start.
- Empty needles, invalid codepoints, negative starts and starts beyond the end retain Java behavior. Null checks still occur before early-return cases.
- Invalid raw Go UTF-8 strings use the original materializing string-search implementation. This preserves existing noncanonical input behavior. Java Strings containing isolated surrogate units remain a separate String storage ABI gap; this experiment verifies surrogate-valued **integer needles**, not unsupported isolated-surrogate String storage.

## Evidence

JDK 21.0.6+8-LTS-188 and Go 1.27.1 darwin/arm64. Java and both Go variants produce identical output for 7,956 cases, each checking indexOf and lastIndexOf: 15,912 method results per implementation. The matrix spans 12 haystacks, 15 string needles, 24 integer needles, default starts and 16 explicit starts including both int32 extremes. It includes BMP, supplementary scalars, isolated high/low surrogate integer needles, repeated/absent needles, embedded NUL, empty strings and low-surrogate starting positions.

Four additional Java exception cases match both Go variants: null receiver, null string needle with negative backward start, null boxed start and null receiver with invalid codepoint. Both complete copied-runtime test suites pass. A separate 120-result invalid-UTF-8 Go regression matches baseline and candidate; it is not a Java storage parity claim.

Allocation diagnostics use three fresh processes per variant, 1,000-call `testing.AllocsPerRun` measurements and separate 1,000-call `runtime.MemStats.TotalAlloc` deltas. Inputs and interface arguments are constructed outside measured calls; a sink retains results. Builds are ordinary optimized non-race builds. `GOMAXPROCS=2`; AllocsPerRun temporarily uses one P. Other campaign processes were active, so no execution timing is collected or inferred. Per-process allocation counters do not aggregate other processes' heaps.

| Probe | Baseline allocations/call | Candidate allocations/call | Baseline bytes/call range | Candidate bytes/call range |
| --- | ---: | ---: | ---: | ---: |
| Direct ASCII delimiter | 3 | 0 | 84 | 0 |
| Actual generated delimiter call shape | 4 | 1 | 100–100.016 | 16 |
| 4 KiB ASCII delimiter | 3 | 0 | 18452.064–18452.080 | 0–0.016 |
| Supplementary integer needle | 3 | 0 | 104 | 0 |
| Isolated high-surrogate integer needle | 3 | 0 | 68–68.016 | 0 |
| Isolated low-surrogate integer needle | 3 | 0 | 68 | 0 |
| Unicode string, forward low-unit start | 4 | 0 | 96 | 0 |
| Unicode string, backward low-unit start | 4 | 0 | 96 | 0 |
| Empty needle, start beyond end | 2 | 0 | 64 | 0 |
| Negative backward start | 4 | 0 | 96 | 0 |
| Invalid codepoint | 2 | 0 | 64 | 0 |

All three AllocsPerRun values agree for every row. The occasional 0.016 bytes/call is 16 incidental process bytes across 1,000 calls; it is retained in `allocation_evidence.json`, not rounded into a false exact total-byte result. No Java allocation measurement was made. The JVM is the independent behavior oracle, not an allocation comparator here.

## Recommendation and remaining validation

Prioritize the ASCII integer search and concrete-string null check: low implementation scope, directly observed generated use, and measured removal of receiver-sized temporary allocation. Then consider streaming integer codepoint and valid-UTF-8 string paths with the supplied oracle matrix. Their UTF-16 boundary handling has greater semantic risk. Keep the invalid-UTF-8 fallback unless the runtime formally excludes such values.

Removing the remaining generated outer boxing requires a separate codegen/runtime decision: a concrete-string null-check helper or an equivalently preserving emitted guard. Do not simply omit receiver validation: Java receiver/argument evaluation and exception ordering require their own parity tests. This experiment makes no compiler change.

CPU gains remain unmeasured. UTF-8 validation and UTF-16 prefix counting add scans, and backward codepoint search can scan twice. Compare baseline, candidate and Java in a coordinated quiet window before selecting all fast paths: mixed ASCII/BMP/supplementary inputs; hit/miss, early/late, short/long needles; default and explicit starts. Use three or more process forks, warmup and repeated measured batches in the same JVM, exact checksum parity every batch, matched CPU limits and documented heaps. Report startup separately and retain per-fork uncertainty. Re-run complete runtime tests and generated application parity after any production adaptation; allocation reduction alone is not evidence of an application speedup.

## Reproduce

From the repository root, with `go version` reporting Go 1.27.1:

```sh
python3 performance/codegen/string_search/prepare.py /tmp/java2go-string-search-repro
python3 performance/codegen/string_search/run.py /tmp/java2go-string-search-repro
```

The destination must be empty. Preparation freezes one runtime snapshot before creating both variants. The runner uses `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/`, writes exact JVM/Go stdout, compiler escape logs, full-runtime test results and all three raw trials into the scratch directory, and fails on parity/test disagreement. The reviewed final run is `/tmp/java2go-string-search-final`. Compact allocation evidence is retained here; verbose logs remain in scratch. Candidate formatting after this run changes whitespace only.
