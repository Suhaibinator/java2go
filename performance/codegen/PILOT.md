# Allocation pilot: generated Go is slower under this configuration

The authorized three-pair pilot completed on 2026-09-27. All **18 full application executions** (nine Java, nine Go, including warmups) produced the exact checked-in allocation-fixture output. Generated Go did not outperform Java: paired median ratios were **2.560, 1.547, 1.549**, giving a geometric Go/Java ratio of **1.831**. This is a noisy exploratory result, not a calibrated claim that Go is universally 83% slower.

Each independent runtime process called the original application main three times: one untimed warmup followed by two timed repetitions **inside the same JVM or Go process**. Ordering was Java/Go, Go/Java, Java/Go. No outlier was discarded or rerun. Startup was not isolated in this pilot; the complete process durations include startup, all three calls and shutdown and are not throughput samples.

| Pair | Runtime | Warmup seconds | Measured seconds | Fork median seconds | Peak sampled RSS MiB |
|---|---|---:|---:|---:|---:|
| 1 | java | 11.644 | 11.572, 13.035 | 12.303 | 238.2 |
| 1 | go | 23.436 | 22.686, 40.318 | 31.502 | 19.2 |
| 2 | go | 19.295 | 18.854, 19.370 | 19.112 | 17.8 |
| 2 | java | 12.299 | 12.327, 12.379 | 12.353 | 222.4 |
| 3 | java | 12.036 | 12.577, 12.094 | 12.336 | 194.3 |
| 3 | go | 20.721 | 19.164, 19.055 | 19.110 | 15.3 |

The 3-pair bootstrap interval is 1.547–2.560. With only three independent pairs and recorded competing CPU work, it is a rough description of this pilot's uncertainty, not a robust confidence statement. One warmup also does not prove fully settled JVM compilation. The conspicuous 40.318s Go sample remains in every summary. The other two pair ratios agree around 1.55, but selecting only those pairs as the headline would hide observed noise.

## Configuration and validity

- Java 21.0.6+8-LTS-188, explicit `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/java`; Go 1.27.1 darwin/arm64.
- Java `-XX:ActiveProcessorCount=2 -Xms32m -Xmx512m`; Go `GOMAXPROCS=2 GOMEMLIMIT=512MiB`, default GOGC. These match active runtime processor counts and nominal memory targets, **not** GC policy or OS affinity. Go's limit is soft; Java Xmx limits heap, not total RSS. Both used a shared 2GiB sampled-RSS abort threshold (~250ms polling), not a hard OS memory cap. Sampled RSS may miss brief peaks.
- Same original allocation source/input and expected stdout. Every warmup and measured invocation was retained in output and validated: each process's stdout had to equal exactly three copies of the oracle; stderr had to contain exactly the expected indexed timing frames and nothing else. All six processes exited zero.
- Mains reset workload state on every call. Original printing is included in timings. Java reflection invocation overhead is included, insignificant at this workload scale but not appropriate for submillisecond kernel comparison.
- A contention snapshot recorded another campaign transpiler at 101.6% CPU, benchmark JVM at 99.8%, WindowServer at 44.2%, plus browser/app activity. Repair agents were authorized to run short tests. There was no dedicated quiet host or fixed CPU affinity. The snapshot does not identify the precise cause of any individual outlier.
- Runtime/transpiler repairs continued during the pilot. The actual compiled class files and Go executable were fixed before measurement and retained; later source edits did not modify these artifacts. The source manifest is a **preparation snapshot**, not an assertion that all concurrently edited source files matched it at every instant of compilation. No compiler changes were made by this performance agent.

## Highest-value next experiments

1. **Make primitive-array writes inlineable, then inspect BCE.** The compiled generic `PrimitiveArrayAssign` body has cost84 against Go's inline budget80; its `primitiveArrayIndex` already inlines with cost64. The fixture's `ScratchRecord` path alone makes 35,389,440 ×28 ×3 = **2,972,712,960** generated calls to the write helper (two constructor writes plus one consume write per word). Move exceptional construction into a cold helper or otherwise simplify the success path without changing null/bounds/RHS ordering. A tiny source change could unlock caller optimization, but no speedup has been measured. This is the first small, general candidate to test.
2. **Guarded direct-slice loops for freshly allocated equal-length arrays.** Constructor bounds and payload length are especially easy to prove; consumer retains three read bounds checks and the checked writes. Preserve zero-trip/null behavior, int overflow, aliasing, side-effect order and fallback. Validate the assignment-timing and primitive-array exception regressions as well as this oracle. Unlike unrestricted bounds-check removal, this offers a general semantic proof.
3. **Reduce array-object allocation layers where nonescape/identity proof permits.** Compiler diagnostics show both `PrimitiveArray` wrappers and their backing slices escape; each record currently has five logical allocation sites (record + two wrappers + two backing arrays), while source Java requests three objects/arrays. Investigate owner-embedded wrappers or elimination only for proven private, nonescaping arrays; reflection/identity, mutation and retained-owner lifetime must remain valid. Do not infer total allocation count from panic-path diagnostics.
4. **Specialize exact local reference-array stores/reads.** `ObjectView` cost1193, `ReferenceArrayGet`145 and `ReferenceArrayAssign`211 are noninline; batches hold known fresh exact `ScratchRecord` objects but use erased storage and checked views. Keep covariance and null normalization, and retain the checked path where proofs fail. Profile first to establish how much of this fixture's CPU belongs to these helpers.
5. **Measure GC policy sensitivity separately.** Go's 15–19MiB sampled RSS versus Java's 194–238MiB is a large policy/footprint difference despite the same nominal 512MiB target. A future GOGC sensitivity sweep under the same RSS ceiling could trade unused memory for throughput. RSS alone does not establish GC as the dominant bottleneck; collect Go CPU/allocation/GC profiles in a separately authorized run. Keep default-policy and tuned-policy comparisons distinct.

These are ranked candidate explanations, supported by compiler/code structure rather than measured CPU attribution. Even the steadier pairs imply a substantial gap; no single proposed change is promised to make Go faster than Java. Execution-token allocations are not a target here: the compiler explicitly reports that they do not escape in these methods. String-loop optimization remains important for different workloads but cannot explain this integer/array fixture's central work.

## Evidence and reproduction

Compact tooling remains in `performance/codegen/`: `RepeatMain.java`, `make_repeat_driver.py`, `paired_runs.py`, and `AUDIT.md`. Verbose compiler evidence moved to `.campaign/performance/codegen-allocation-compiler.log`; raw pilot artifacts are under `.campaign/performance/codegen-allocation-pilot/` (ignored by git): `config.json`, `results.json`, exact stdout/stderr per process, generated sources, Java classes, fixed Go executable, version/hash metadata, preparation source hashes, source-change context, and contention snapshot.

The retained executable's SHA-256 is `1f4b8d5017aa89bd0bf56ca9fe8fefb121b30e6a37044256c199173004dd5f2b`; oracle SHA-256 is `2bdffcb7ee1e1cf7a4e145fe34c877f3b8bdea63c803876b8ed983759651d3ac`. Base git HEAD was `c1f1ae45cfe8d5465425efc562a54580add4f3ea` plus concurrent campaign edits. Fixed-artifact reproduction after reserving another slot:

```sh
python3 performance/codegen/paired_runs.py   .campaign/performance/codegen-allocation-pilot/config.json   .campaign/performance/codegen-allocation-pilot/repeat-results.json
```

The archived config contains exact argv arrays, absolute artifact paths and environment overrides. To rebuild after source changes, use the generation/compiler commands in `AUDIT.md`, compile `RepeatMain.java` with the fixture's original Java sources using `javac --release 21 -encoding UTF-8`, and generate/build `repeat/main.go` with import `parity/allocation/app`. Treat that rebuild as a new baseline; do not merge its measurements with this fixed-artifact pilot. Change the output and artifact-directory paths to preserve existing evidence.

Pilot elapsed process time totaled approximately 313 seconds. No second suite, startup comparison, CPU profile, optimization or outlier retry was performed.
