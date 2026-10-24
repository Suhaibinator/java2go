# Correctness-matched sustained benchmark design (not executed)

Use the accepted complete data application, its frozen real Commons Codec dependency, and the original 1,300-byte resources. Keep all Unicode, malformed records, stream tracking, replay state and output files. Grow duration by repeating complete independent jobs; do not enlarge or simplify the accepted data, bypass dependencies, embed oracle outputs, or hand-edit generated implementations.

## Workload and builds

1. Snapshot original Java sources/resources, dependency source closure/JAR hash, transpiler/runtime commit and toolchains. Build non-race, non-profiled release artifacts. Existing accepted race-run durations are excluded. Java comparator is the pinned binary dependency; also rerun source-closure Java parity once to preserve the acceptance premise.
2. Add separate harness mains only: Java calls the unchanged `campaign.data.app.Main.main(String[])` directly; Go imports the unchanged generated app and calls `Java2goEntryMain` with its normal ReferenceArray arguments. Preserve normal resource registration. The accepted generated Go implementation is immutable in both baseline and any runtime-candidate copy.
3. One batch has a fixed externally selected number K of jobs, cycling seeds 17, 41, 97 in exactly that order, with K divisible by 3. Prepare all argument arrays and unique output-directory paths before the timer. Time only the loop calling original mains. Every call writes its original stdout and all three normal files. Use a distinct output directory per job so a later job cannot hide an earlier file discrepancy.
4. After the batch timer stops, verify every job's stdout segment, zero application stderr, and SHA-256 of accounts.tsv/summary.txt/rejects.tsv against a fresh live Java oracle. Compare those live oracles with frozen accepted outputs before timing. Output directories can be reclaimed only after verification and outside timers. Capture command exit status and timeouts; reject the whole result on any mismatch. Harness timing/manifest records go to a separate descriptor or structured file so they cannot hide application stderr.
5. Use identical K, seeds, argument/path lengths, filesystem, resource placement, output sink and validation schedule for JVM/Go. Directory creation and actual file writes are part of application time. Validation happens outside the timed batch but can affect GC/cache state; disclose it. Do not call this pure CPU throughput. A later component benchmark must be separately defined and parity-gated, never substituted silently for the complete application.

## Startup and warmed execution

- Startup: separately run one ordinary precompiled application process per seed, with original behavior, and measure external launch-to-exit latency. Randomize/alternate runtime order and verify stdout/files. Do not mix these numbers with same-process batches. Compilation and dependency resolution are excluded.
- Calibration: in a coordinator-approved quiet window, try a small K (for example 30) and increase a fixed K until a batch is approximately 1–3 seconds on the slower runtime. Freeze that same K for both. Pilot calibrations are not headline samples.
- Warmed throughput: each fresh JVM/Go fork runs at least 3 complete untimed batches and then 5 measured batches in that **same process**. Raise warmup duration if JVM compilation/throughput trends are still changing; require at least about 10 seconds total warmup before interpreting steady-state. Validation still applies to every warmup job. Never use a separate-process warmup as evidence the measured JVM is warm.
- Start with 3 alternating independent fork pairs as a pilot. Once stable and approved, use 8 independent pairs; report all batches, per-fork medians, paired Go/Java geometric ratio, fork-level bootstrap interval, and ranges. Do not treat 5 within-fork batches as 5 independent processes or retry/discard slow runs solely to improve a ratio.

## Resource controls and measurements

Use the same host, power/thermal state and no competing compiler/test runs. Set Java ActiveProcessorCount=2 and Go GOMAXPROCS=2. These do not impose hard OS CPU affinity; document macOS scheduling limitations. Record Java heap policy and Go GOGC/GOMEMLIMIT separately. Equal numeric Xmx/GOMEMLIMIT values do not imply equal GC policy or hard total-memory caps. Use a shared 2 GiB sampled RSS guard for the initial bounded pilot and record sampled peak RSS; escalate the common budget only if correctness/working-set demands it, before a new series.

Allocation profiles in this directory used MemProfileRate=1 and forced GCs and must never supply headline timings. Collect Go CPU/allocation profiles and JVM JFR/GC/compiler diagnostics in separate parity-verified runs. Use those to attribute costs; do not infer CPU dominance from allocated bytes or compare Go Mallocs directly with JVM sampled allocation events.

## Candidate A/B gate

First proposed candidate: general allocation-free UTF-16 String comparison (or narrower ASCII guard), followed by general ASCII substring extraction. Validate null/index exception timing, exact comparator differences, BMP/supplementary ordering, stable sorting and the full accepted data oracle. Keep dependency/application/generated implementations byte-identical. Compare Go baseline versus candidate with the same executable harness, then compare each with the same JVM build in the same scheduled session. Only those paired verified results could support a claim of outperforming Java.

No startup/sustained Java-versus-Go measurement or candidate speedup claim has been made for this accepted data application.
