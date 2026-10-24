# Stateful UTF-16 StringBuilder allocation investigation

The generated Go baseline allocates substantially more heap bytes than Java in this seeded text-editing workload. Two isolated runtime candidates reduce those bytes, but **neither beats Java's allocation result**. The best candidate still allocates about **2.31× Java for ASCII, 2.10× for mixed Unicode, and 1.92× when taking a snapshot after every edit**. No execution time was collected or inferred, and no production source was changed.

The compiler and runtime are frozen from pushed revision `8cb6cd5` through a read-only archive. Source, generated files, snapshot manifest and candidate hashes are recorded in `allocation_evidence.json`; the complete manifest and raw logs are in `/tmp/java2go-stringbuilder-verified`. This avoids using active campaign edits. The first preparation attempt encountered an incomplete live compiler change; it contributes no measurements. An exploratory Java `String.hashCode()` consumer also emitted an unsupported Go call at the frozen revision. The final Java workload uses an explicit UTF-16 checksum and passes parity before all measurements.

## Workload and correctness

`BuilderWorkload.java` is compiled by JDK 21 and transpiled unchanged. Each run builds 16 texts through 128 seeded stateful edits each, using four seeds (`1`, `17`, `9031`, `-104729`). Operations include string, integer, char and char-array append; front string/char insertion; end char-array insertion; reverse; and deletion of a temporarily appended ASCII unit. Supplementary characters enter both as strings and as individual high/low char appends. No completed String contains an isolated surrogate.

The modes are ASCII with snapshots every 16 edits, mixed BMP/supplementary text with snapshots every 16 edits, and that same Unicode workload with snapshots every edit. Every snapshot's full UTF-16 contents feed the checksum through Java `toCharArray()`. A state-dependent builder `charAt` contributes after every edit, and all final builder units contribute too. This measures a text builder plus a real consuming checksum; it is not a pure append microbenchmark. The frequent-snapshot mode intentionally exposes repeated representation conversions.

All 12 complete seed/mode stdout lines match exactly between the original JVM program, baseline generated Go, and both isolated candidate runtimes. Generated application files are hashed before every variant build and remain identical. All 576 measured runs (12 cases × 4 repetitions × 3 fresh forks × 4 variants) are checked against the independent Java output. Warmup repetitions check consistency, and each profiled run also checks its checksum. Full copied `stdjava` test suites pass for baseline and both candidates. The separately run existing JVM-backed `TestCampaignRuntimeStringBuilderUTF16` also passes for all three runtime copies, covering surrogate-pair reverse, temporary isolated-unit editing and bounds exception type/message parity. Builder-specific null and alias combinations remain additional production-adoption checks. A checksum cannot formally establish equivalence for every possible input; the supplied results cover this workload and those runtime tests.

## Measured allocations

Environment: JDK 21.0.6+8-LTS-188, Go 1.27.1 darwin/arm64. Java uses `-XX:ActiveProcessorCount=2 -Xms64m -Xmx512m`, default compact strings enabled. Go uses `GOMAXPROCS=2 GOMEMLIMIT=512MiB GOGC=100`; its memory limit is a soft runtime target, unlike Java's heap maximum. Both use ordinary optimized non-race builds. Each fresh process performs an initial reference run, then 12 same-process warmups and four measured runs per case. Three process forks are interleaved by runtime. The JVM is never restarted between warmup and measurement.

Java reports current-thread allocated bytes through `com.sun.management.ThreadMXBean`; Go reports process `runtime.MemStats.TotalAlloc` and `Mallocs` deltas. These measure allocated heap bytes, not live heap, RSS, GC cost or time. The Go counter includes runtime/background allocations whereas the Java counter is scoped to the workload thread; small differences must not be overinterpreted. Other campaign processes were active, so no throughput comparison is attempted. Memory counters do not aggregate another process's allocations.

Median bytes per run across the 48 observations for each mode (4 seeds × 4 repetitions × 3 forks):

| Runtime / candidate | ASCII | Mixed Unicode | Unicode snapshot every edit |
| --- | ---: | ---: | ---: |
| JDK 21 original source | 134,428 | 248,664 | 2,487,972 |
| Generated Go, baseline | 895,312 | 1,127,312 | 6,532,116 |
| Go, direct append + insertion shift | 455,856 | 630,952 | 6,058,700 |
| Go, above + direct UTF-8 output | 309,944 | 521,448 | 4,766,336 |
| Baseline Go / Java, allocated bytes | 6.66× | 4.53× | 2.63× |
| Best candidate / Java, allocated bytes | 2.31× | 2.10× | 1.92× |

Across all seeds and repetitions, baseline Go ranges were 875,856–905,568; 1,088,176–1,141,272; and 6,447,592–6,584,200 bytes. Java ranges were 129,416–136,272; 245,096–249,576; and 2,444,880–2,501,784 bytes. Most range width comes from different seeded edits, not measurement uncertainty. Per-seed, per-fork raw values and each fork's median are retained, rather than presenting these ranges as confidence intervals. Java's three fork medians were identical; Go's medians have small process-level variation.

## What the profiles explain

Separate diagnostic processes set `runtime.MemProfileRate=1` and use forced-GC profile deltas. They are attribution evidence only; profile accounting may lag GC and is not substituted for the ordinary allocation-counter measurements. Examples below use seed 1.

- Baseline mixed mode allocates approximately 467 KB across the principal `StringBuilder.Insert` growth sites, including `InsertChar`. Current insertion copies the entire tail even when retained capacity could support an overlapping in-place shift.
- Baseline mixed mode's snapshot consumer allocates 256,080 bytes in `StringChars` through `StringToCharArray`. Builder output additionally allocates 161,840 bytes in its temporary decoded rune slice and 63,328 bytes in rune-to-string conversion.
- With a snapshot every edit, `StringChars` through `StringToCharArray` accounts for 3,051,984 attributed bytes. The temporary output rune slice accounts for another 1,937,600. These dominate the edit cost.
- The direct-output candidate removes that temporary decoded slice, but `strings.Builder.Grow(len(UTF16Units))` underestimates UTF-8 output size for much of the Unicode text. Subsequent `WriteRune` growth remains visible: about 916 KB across the two main growth sites in frequent-snapshot mode. This candidate reduces bytes without consistently reducing the number of output allocations.
- Generated string-interface conversions and char-array wrappers are smaller but real: mixed-mode profiles show 22,992 bytes at generated `convTstring` sites, 6,600 at `InsertChars` slice boxing and 6,408 at `AppendChars` slice boxing. Numeric append emits `Append(StringValueOf(step))`, incurring formatting before UTF-16 append; `fmt.Sprint` contributes about 4,944 bytes in this case.

The generated code confirms the paths: `Append(StringValueOf(word))`, `Insert(0, StringValueOf(word))`, `AppendChars`, `InsertChars`, and `StringToCharArray(StringRequireNonNull(value))`. The unit checksum uses primitive-array iteration directly. Generated evaluation-order helpers remain unchanged.

## Isolated candidates and recommendation

`candidates.py` builds two runtime copies; zero-context `edits.patch` and `edits_output.patch` show the corresponding changes against the frozen baseline. These are reviewable prototypes, not proposed acceptance outputs or live fixes. Format with `gofmt` if adapting them for production.

| Priority | Opportunity | Evidence and prerequisites |
| --- | --- | --- |
| 1 | Append string units directly; grow then shift insertion tail in place | The `edits` candidate cuts ordinary-mode allocation by roughly 44–49%. Append and insertion changes were measured together, so do not assign the entire reduction to either alone. Keep offset validation and surrogate units intact; retain char-array input independence and overlapping-copy correctness. |
| 2 | Encode builder output directly into UTF-8 | Additional measured byte reduction, strongest with repeated snapshots. Preserve rejection of isolated surrogate String output and immutable snapshot independence. An exact UTF-8 capacity pass could avoid the observed reallocation; it trades extra scanning against allocation and needs its own CPU/byte measurement. |
| 3 | Reduce String-to-char-array overcapacity and the four-byte unit representation | The largest remaining frequent-snapshot allocation is the consumer conversion. `StringChars` reserves `len(UTF8Bytes)` rune elements, often more than the UTF-16 unit count, and each stored unit occupies four bytes. Exact unit sizing is a narrower experiment; a `uint16` builder/char-array representation has broader ABI and conversion risk. Neither has been implemented here. |
| 4 | Add typed append/insert/null-check paths where codegen already knows the type | Profiles show interface boxing and numeric formatting. Preserve Java overload selection, null-to-text behavior, array null checks and argument evaluation order. The expected benefit is smaller than the bulk buffers in these workloads; no speedup estimate is justified. |

Inspection of this installed JDK's `lib/src.zip` (`java.base/java/lang/AbstractStringBuilder.java` and `StringBuilder.java`) explains why Java remains difficult to beat: it stores builder contents in a coder-tagged byte array, uses one byte per Latin-1 unit or two per UTF-16 unit, copies string contents directly into retained capacity, and shifts insertion contents within that capacity. The Go builder stores UTF-16 units as four-byte runes and moves through UTF-8 Strings for each snapshot. Its `Reverse` also always makes a second surrogate-repair scan; Java has a Latin-1 path. A tracked “may contain surrogates” property could skip that second pass, but this has not been tested and adds mutation-state obligations.

The workload establishes allocation losses and concrete opportunities; it does not establish that Go is slower or that either candidate makes an application faster. Before any runtime change, adapt candidates separately with targeted insert/array alias/null/bounds and snapshot-immutability checks, then run runtime and generated application parity. For a CPU comparison, reserve a quiet window, use the same original source and seeds, warm within each JVM process, retain per-fork uncertainty, and separate startup from sustained execution. Do not time the profile builds.

## Reproduction

From the repository root with Go 1.27.1 and the JDK installed at `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`:

```sh
python3 performance/codegen/stringbuilder/prepare.py /tmp/java2go-builder-repro
python3 performance/codegen/stringbuilder/run.py /tmp/java2go-builder-repro
```

Use a new empty scratch directory. Preparation reads `8cb6cd5`, builds the frozen compiler, compiles/transpiles the same Java source and requires exact parity. The runner creates candidate runtime copies, verifies unchanged generated sources and both candidates' output, runs full runtime suites and the JVM-backed UTF-16 builder regression, and only then collects allocation counters and profiles. It fails on every detected parity/test mismatch. No generated application code is edited. All logs and full raw evidence remain in scratch; compact reviewed evidence is retained beside this report.
