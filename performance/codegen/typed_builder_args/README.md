# Typed StringBuilder arguments: codegen allocation experiment

**Known-String argument boxing creates many small objects, but UTF-16/String buffers dominate allocated bytes.** A general isolated compiler candidate removes about 36–37% of allocation events in ordinary stateful editing while reducing bytes by only 1.9–2.2%. With a String snapshot after every edit, the byte reduction is 0.33%. This supplies a bounded codegen opportunity; it does not close the allocation gap to Java or establish a speedup.

This experiment uses the unchanged `../stringbuilder/BuilderWorkload.java` from the committed earlier investigation, with compiler/runtime sources frozen at `8cb6cd5`. Both compilers independently regenerate it. No generated application Go is edited or substituted as acceptance evidence. All changes are confined to this artifact directory and `/tmp/java2go-typed-builder-verified`.

## Candidate and scope

The compiler's existing builder intrinsic emits `text.Append(StringValueOf(word))` and `text.Insert(0, StringValueOf(word))` even when `word` is statically a Java String. That boxes the string for generic text conversion, formats it through `fmt.Sprint`, then boxes the resulting string for the generic builder helper.

`candidate.patch` contains a five-line general lowering addition and two typed runtime helpers. Statically built-in String arguments become `AppendString` / `InsertString`; other overloads retain their previous lowering. A user-defined type named String is excluded through the existing symbol resolver. Nullable interface-backed String locals go through existing `coerceArgumentToExpectedType`, preserving the concrete null sentinel. The typed helpers turn that sentinel into the text `"null"`.

Crucially, the helpers retain the baseline `StringChars` materialization, insertion-tail copy, buffer representation and return semantics. Snapshot encoding and `StringToCharArray` are unchanged. This separates argument boxing/formatting from the larger buffer improvements studied previously. The shared StringBuffer intrinsic also uses this lowering; its existing runtime synchronization limitations are outside this prototype.

The first isolated attempt omitted nullable-storage coercion and failed Go compilation on the supplemental oracle's `String missing = null` local. It was corrected using the compiler's existing conversion machinery before any allocation measurement. This failure is why Java static type alone is insufficient to choose a Go parameter type.

## Correctness and validation limits

The unchanged workload has four seeds and three modes, each with 16 builders and 128 stateful edits per builder. Modes are ASCII, mixed BMP/supplementary text, and mixed text with a snapshot after every edit. It includes string/char-array/integer/char append, insertion, reverse and deletion; every complete snapshot contributes all UTF-16 units to the checksum. The source contains no isolated-surrogate String output.

Fresh JDK 21 executions produce the full observations in `BuilderWorkload-java.stdout` and `ArgumentOracle-java.stdout`. They are retained as evidence, not embedded into application source. Both independently generated Go variants match those observations exactly before measurement. The supplemental oracle checks nullable locals, explicitly cast nulls, qualified String types, empty/supplementary/BMP strings, receiver→offset→argument evaluation order, bounds failure after argument evaluation, erased Object fallback, unaffected numeric/char overloads and snapshot independence. It temporarily inserts a character inside a surrogate pair and repairs the pair before String output.

All 288 measured runs and six separately profiled runs match the fresh Java workload observations. Both complete copied `stdjava` suites and these five generated-behavior tests pass:

- `TestRuntime_StringBuilder`
- `TestRuntime_StringBuilderOverloadText`
- `TestCampaignRuntimeStringBuilderUTF16`
- `TestCampaignRuntimeStringBuilderChain`
- `TestNullableStringReferencesPreserveJavaSemantics`

**This is not a full compiler-suite pass.** The existing `TestIntrinsics_StringBuilder` rendering test fails on its literal expectation `sb.Append("a")`; the candidate correctly emits `sb.AppendString("a")` instead (and `InsertString` for its String insert). The failure is recorded separately and the original test/oracle was not edited. A production adaptation must update the affected source-shape assertions, add targeted typed-lowering cases, and run the full required checks. The five behavior tests and fresh JVM observations establish only their tested scope.

## Allocation results

Go 1.27.1 darwin/arm64; optimized non-race builds with compiler diagnostics enabled but no optimization-disabling flags. Each of three fresh forks performs an initial reference run, 12 same-process warmups and four measured runs per seed/mode. Baseline and typed processes are interleaved. `GOMAXPROCS=2`, `GOMEMLIMIT=512MiB`, `GOGC=100`; the host remains contended by the campaign. `runtime.MemStats` supplies process heap allocation counters. The memory limit is soft, and runtime/background allocations can contribute small variations.

The table reports medians across 48 observations per mode. Allocations can have fractional medians across different seeds; no individual run allocates a fractional object.

| Mode | Baseline bytes/run | Typed bytes/run | Byte reduction | Baseline allocations/run | Typed allocations/run | Allocation-count reduction |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| ASCII, periodic snapshots | 895,232 | 875,592 | 2.19% | 4,256 | 2,680 | 37.03% |
| Mixed Unicode, periodic snapshots | 1,127,320 | 1,105,648 | 1.92% | 4,382.5 | 2,804.5 | 36.01% |
| Mixed Unicode, every-edit snapshots | 6,532,228 | 6,510,588 | 0.33% | 17,815.5 | 16,238 | 8.86% |

The three baseline fork medians for mixed mode are 1,127,312 / 1,127,320 / 1,127,312 bytes, versus 1,105,640 / 1,105,648 / 1,105,640 for typed. Every per-seed repetition and all fork medians are in `allocation_evidence.json`. Seeds produce different work distributions, so a pooled range is not a statistical confidence interval.

No Java allocation trial was repeated: the earlier investigation already measured that identical Java workload using same-process JDK warmups. Its allocated-byte medians were 134,428 / 248,664 / 2,487,972. Those prior JVM values remain far below this typed-only Go candidate. This is historical allocation context, not a newly paired throughput comparison. No timing was collected for either runtime, and no claim about latency, throughput or GC time follows from these counts.

## Attribution

Separate processes use `MemProfileRate=1` and forced-GC profile deltas, with checksum validation. These diagnostics attribute bytes and objects; GC/profile accounting may lag, so the regular MemStats trials provide the comparison above.

For seed 1 in mixed mode:

- Generated `convTstring` sites fall from **22,992 bytes / 1,437 objects** to **6,000 bytes / 375 objects**.
- `fmt.Sprint` under `StringValueOf` falls from **4,944 bytes / 309 objects** to **512 bytes / 32 objects**. Numeric and erased-reference paths deliberately remain generic.
- The String append/insert `StringChars` sites remain **2,976 / 2,880 bytes** in both variants; their stack names change to the typed methods. This confirms their materialization algorithms were retained.
- The large insertion-tail site remains **254,800 bytes** in both variants. Other insertion and char-array-wrapper costs also remain.

For frequent snapshots, the largest conversion sites are unchanged: **3,051,984 bytes** in `StringChars` through `StringToCharArray`, **1,937,600 bytes** in the intermediate decoded rune slice in `StringFromChars`, and about **756,500 bytes** for the final UTF-8 String. Together those three sites are about **88%** of baseline attributed bytes. Removing String argument boxing therefore cannot substantially reduce this mode's byte volume.

`compiler_evidence.txt` records that typed String parameters do not escape. The helpers themselves are **not inlined** (reported costs 86 and 171, over the budget of 80). The observed allocation reduction is not evidence of call inlining or a measured CPU gain.

## Recommendation

The typed String overload is a useful small general codegen candidate, particularly if allocation count or formatting CPU becomes a demonstrated bottleneck. It handles String arguments without runtime type dispatch while preserving erased Object and primitive behavior. Its byte savings are modest for this workload, so prioritize snapshot conversion and insertion-buffer improvements when the goal is to reduce total allocated memory.

Keep changes separate during production review: typed argument lowering, String output encoding, String-to-char-array sizing and insertion shift each have different semantic risks and can be validated independently. For this codegen change, validate nullable storage forms, all String expression shapes, shadowed type names, StringBuffer, side-effect ordering and overload preservation; then run the full compiler/runtime/application gates. The supplied patch is an isolated prototype, not an accepted production change.

A future CPU comparison needs a coordinated quiet window, same-process JVM warmup, matched workload inputs, multiple process forks and complete parity per repetition. Fewer objects can plausibly reduce allocator/formatting work, but no speedup percentage is inferred here.

## Reproduce

The scripts require the earlier authenticated frozen experiment directory, including its `metadata.json` and `snapshot/`, rather than reading an actively changing worktree or invoking Git. The default is `/tmp/java2go-stringbuilder-verified`; pass `--frozen PATH` for another matching copy. Every original snapshot file is verified against its recorded hash before copying. Go 1.27.1 and `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home` are required.

```sh
python3 performance/codegen/typed_builder_args/prepare.py /tmp/java2go-typed-builder-repro
python3 performance/codegen/typed_builder_args/validate.py /tmp/java2go-typed-builder-repro
python3 performance/codegen/typed_builder_args/measure.py /tmp/java2go-typed-builder-repro
```

Use a new empty destination. Preparation recompiles both compilers and generates both programs from unchanged Java. Validation builds the separate allocation driver and runs the listed semantic checks. Measurement checks every result against fresh JDK output and rechecks generated-file hashes afterward. The known rendering-test failure is documented separately, outside those passing behavior gates. Full raw compiler logs, observations, test logs and measurements remain in `/tmp/java2go-typed-builder-verified`; reviewed evidence is retained here. The helper artifact was gofmt-formatted after the experiment and already matches the formatted candidate source used during measurement.
