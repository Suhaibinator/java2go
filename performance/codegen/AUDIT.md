# Generated Go CPU/allocation audit

Snapshot: 2026-09-27, base HEAD `c1f1ae45cfe8d5465425efc562a54580add4f3ea`, with concurrent campaign edits. Host compiler: Go 1.27.1 darwin/arm64. Java target for new comparisons: `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`. This report makes **no new measured Java-versus-Go performance claim**. Compiler evidence is structural, not timing. Performance work must use a frozen passing campaign snapshot before execution.

## Findings and ranked opportunities

| Priority | General optimization | Concrete evidence | Expected value / risk | Prerequisites and validation |
|---|---|---|---|---|
| 1 | Prove canonical primitive-array loops once, use local native slice views in a guarded fast loop | Fresh generated allocation `ScratchRecord.ConsumeJava2goExecution` has `left.Elements[index]`, `right.Elements[mirror]`, `right.Elements[index]`; compiler reports three `IsInBounds`. Length checks and checked writes remain. `NewScratchRecord` fills two newly created equal-length arrays. | High potential in sustained allocation/integer workloads; medium semantic risk. No speedup estimate. | Start from constructor-local nonescaping arrays/equal lengths. Keep zero-trip/null behavior, signed Java-int overflow, evaluation order, mirrored interval proof, and original fallback. Validate exact integer fixture stdout, null/bounds exception type and timing, array assignment timing fixtures, aliasing, mutation of bounds, and compiler BCE before timing. |
| 2 | Reuse a loop-local UTF-16 view of a stable string; specialize proven ASCII without decoding on every access | `stdjava/strings.go:73` calls `StringChars(s)[index]`; `StringChars` scans the entire string and allocates a rune buffer with capacity `len(s)`; `StringLength` also scans. A normal full charAt traversal costs O(n²) decoding/allocation volume. `StringCompareTo` decodes both complete strings even when first units differ. | High potential in parsing/hash-heavy applications; high semantic risk until current UTF-16 gaps are fixed. | Gate on stable receiver/local scope; preserve laziness and exception timing. Test empty/BMP/supplementary/unpaired surrogate strings, null, negative/end indices, reassignment, observable side effects. Pending UTF-16 correctness repair is a prerequisite. Avoid global caches that retain strings. String hashes can instead stream UTF-16 units without intermediate slices; collision/equality behavior must remain exact. |
| 3 | Specialize reference-array get/store for a locally proven exact element representation | Allocation kernel stores millions of fresh `ScratchRecord`s into fresh exact arrays, but `ReferenceArrayAssign` still performs nominal/covariant machinery and reads call `ObjectView`. Backing storage is `[]any`, doubling element words relative to a pointer slice on arm64. | Potentially high for object-heavy workloads; high ABI/identity risk. | First elide redundant type validation only for exact typed constructor stores into proven array identity, preserving bounds/RHS order. Do not globally replace `[]any` with `[]T`: covariance, erased views, base subobjects, typed nil and reflection matter. Measure helper CPU in a profile before introducing a typed backing representation. Validate collection, boxed-object, covariant-array and generic-dispatch parity fixtures. |
| 4 | Specialize generated-only interface/functional-adapter calls where the concrete implementation is proven | `generated/full_program/domain/ParseTask.go` constructs a generated mapper then type-asserts it to `MapperJava2goExecution` on invocation, with fallback to public wrapper. `common/Mapper.go` stores an indirect function field. Generated interface calls in `transpiler/expression.go` use companion probing. | Medium potential for callback-heavy loops; medium/high dispatch risk. | Keep public Go ABI and foreign implementations supported. Prove exact generated receiver and call execution companion directly; preserve Java argument evaluation before null failure, overriding, generic bridges and source-name collision handling. Verify Go devirtualization/escape diagnostics first: the compiler may already remove some scaffolding. |
| 5 | Reduce codegen expression staging and allocation layers only when compiler evidence demonstrates cost | Allocation consumer cost229 and constructor cost330 exceed inline budget80. Compound assignment staging emits nested closures in generated allocation kernel. Primitive arrays have a wrapper plus backing allocation. | Medium potential; medium/high risk. Surface syntax alone is insufficient. | Use `-m=2`, BCE diagnostics and CPU/alloc profiles to distinguish inlined/nonallocating IIFEs from real overhead. Lower equivalent statements with explicit temporaries where legal, preserve saved LHS/old-value timing. Scalar replacement or owner-embedded wrapper storage requires escape/identity proof and alias tests. |

## Evidence that rules out tempting shortcuts

- Compiler diagnostics archived in `.campaign/performance/codegen-allocation-compiler.log` explicitly report `__java2goExecution does not escape` and wrapper `&stdjava.Execution{} does not escape` throughout the freshly generated allocation model. Removing execution tokens is not supported as a heap-allocation win here. They carry Java logical-thread semantics in monitors/class initialization and cannot be globally shared.
- Panic-path throwable allocations in escape diagnostics are **not** allocations on every successful array access. Do not sum them as normal-path alloc/op.
- Existing matrix affine Tier 1/Tier 2 lowering already caches views, versions null state and hoists interval proofs/row slices. Read `transpiler/affine_array_loop.go` and the numerical fixture README before proposing duplicate work.
- Numerical README records rejected boundary peeling (floating-bit changes), closure-based peeling (slower), neighbor maps (slower), and power-of-two masks (slower with the same hot bounds checks). These are historical experiments on different toolchains, not current measurements.
- Boxing removal requires proving identity is unobserved. Java wrapper caches, `==`, monitors, reflection and exact nominal wrapper types rule out unconditional unboxing into primitives. Prefer generated local arithmetic that is already primitive; do not weaken collection key semantics.
- Historical README times used different JDK/Go versions and noisy sessions. They motivate candidates but establish no current superiority.
- Quantized numerical stdout is a workload oracle, not proof of strict Java floating arithmetic semantics. Any floating rewrite must additionally preserve exact floating bits where the contract requires them, especially given Go contraction versus Java evaluation differences.

## Reproduce the compiler evidence

Run from the repository root. This compiles only; it does not time or claim parity for the new generated snapshot.

```sh
GOCACHE=/tmp/java2go-campaign-go-cache go run ./cmd/java2go -w -strict \
  -output /tmp/java2go-perf-codegen-allocation -module parity/allocation \
  -init-go-mod testfiles/applications/allocation_gc_pressure/src
cd /tmp/java2go-perf-codegen-allocation
go mod edit -require=github.com/NickyBoy89/java2go@v0.0.0 \
  -replace=github.com/NickyBoy89/java2go=/Users/suhaib/.codex/worktrees/adversarial-java/java2go
GOCACHE=/tmp/java2go-campaign-go-cache go build -mod=mod \
  '-gcflags=parity/allocation/model=-m=2 -d=ssa/check_bce/debug=1' ./... \
  > /tmp/java2go-perf-codegen-allocation-compile.log 2>&1
```

Observed important diagnostics: `ScratchRecord.go:22` constructor inline cost330; `:39` consumer cost229; retained index checks at `:46` twice and `:48` once. Source lines describe this generated snapshot and can move after fixes.

## Comparison protocol

1. Root agent reserves a quiet machine slot; no competing builds/tests/benchmarks. Freeze source, generated source, runtime and build flags, record hashes and versions. Re-run each fixture's live Java/generated-Go exact stdout/empty stderr parity first. Do not collect or report timings from wrong-output programs.
2. Startup: fresh processes, already compiled artifacts. `paired_runs.py` with `mode: startup` measures complete process launch/work/shutdown separately. Use a small parity workload for startup sensitivity and the sustained fixtures for total-job latency. Compilation is excluded. Do not call an empty-loop microbenchmark application throughput.
3. Warmed throughput: build `RepeatMain.java` alongside original Java classes; it repeatedly invokes the same original main in one JVM. Generate an equivalent Go repeat driver with `make_repeat_driver.py` inside the generated module. Each fork executes 3 untimed full main warmups and 5 measured full main calls; use at least 8 independent Java/Go fork pairs in alternating order. Increase warmups if warmup/measured trends show ongoing JIT compilation. The 10–14s historical fixtures make 3 warmups a plausible starting point, not a guarantee of steady state. Log actual durations and investigate GC/JIT drift; never discard slow samples merely because they weaken the claim.
4. Both drivers retain every workload result and write the original output each iteration. Runner rejects any process whose full stdout is not exactly the fixture oracle repeated `warmups + samples` times, or whose stderr is not precisely the timing frames. These wrappers are appropriate only for deterministic reentrant mains with no persistent mutable static state, process exits, shutdown hooks or remaining background work. Numerical/integer/allocation mains use local workload state; verify this remains true on the frozen snapshot. Reflection dispatch overhead is included in Java, and per-main output is included in both. For submillisecond workloads, design typed direct kernel drivers instead.
5. Match inputs and active processor budgets (for example Java `-XX:ActiveProcessorCount=4`, Go `GOMAXPROCS=4`), on the same host at the same thermal/power settings. These settings control runtime parallelism; they are not OS CPU affinity. On macOS record the lack of hard affinity. Use the same stated process memory ceiling and monitor actual peak RSS. Java `-Xmx` and Go `GOMEMLIMIT` are different mechanisms: the latter is soft and neither alone caps total RSS. A comparison claiming equal hard memory ceilings needs supervisor enforcement; never claim equivalent memory policy merely from equal numeric values. Run a documented heap/GC-policy sensitivity matrix separately if GC dominates.
6. Report raw timings, per-fork medians, paired geometric Go/Java ratio and bootstrap interval over **independent fork pairs**, plus sample range, peak RSS, allocations and profiles. The runner supplies raw records and exploratory bootstrap interval; 8 forks is a starting point. An interval containing 1 supports no stable advantage. Within-fork iterations are correlated and must not be treated as independent forks.
7. Profile separately from headline timing: Go CPU/alloc pprof with native generated drivers; JVM JFR or compilation/GC logging in separate parity-valid runs. Do not include instrumented data as uninstrumented speed. Confirm allocation count/bytes and hot instructions, not only wall-clock time.

## Harness usage

Compile `RepeatMain.java` with JDK21 into the original Java class directory. Invoke:

```text
java <documented VM flags> -cp <classes> RepeatMain <fixture main class> 3 5
python3 performance/codegen/make_repeat_driver.py parity/allocation/app <generated-module>/repeat/main.go
go build -o <repeat-binary> ./repeat
<repeat-binary> 3 5
```

The `paired_runs.py` config is JSON with `mode`, `java` and `go` argv arrays, `oracle` path, `forks: 8`, `warmups: 3`, `samples: 5`, optional `java_env`/`go_env`, optional working directories, `timeout_seconds`, `rss_abort_bytes`, `artifact_directory`, and a `resources` description. The runner samples child RSS through `ps` about every 250ms and aborts above the configured sampled ceiling; this is not a hard OS cap. `artifact_directory` retains exact validated stdout/stderr. All paths should be absolute. Commands are launched without a shell. Example steady config shape:

```json
{
  "mode": "steady",
  "java": ["/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/java", "-XX:ActiveProcessorCount=4", "-Xmx2g", "-Dfile.encoding=UTF-8", "-Duser.language=en", "-Duser.country=US", "-Duser.timezone=UTC", "-cp", "/absolute/classes", "RepeatMain", "parity.allocation.app.AllocationGcApplication", "3", "5"],
  "go": ["/absolute/generated-repeat", "3", "5"],
  "go_env": {"GOMAXPROCS": "4", "GOMEMLIMIT": "2GiB"},
  "oracle": "/absolute/fixture/expected.stdout",
  "forks": 8,
  "warmups": 3,
  "samples": 5,
  "resources": "4 runtime processors; example heap settings, NOT an equivalent hard RSS cap; arrange shared ceiling externally"
}
```

Run `python3 performance/codegen/paired_runs.py config.json results.json` only after root coordinates the quiet slot. The default 900s process timeout covers the historical sustained fixture scale; adjust before starting if warmup count/work grows.

Validation performed: both scripts parse, Java repeat driver compiled with JDK21, generated Go repeat driver compiled with Go1.27.1, two alternating smoke fork pairs validated repeated exact output. Smoke workload is a print-only synthetic main and its timings have no performance significance. The initial audit did not run sustained benchmarks. A later authorized allocation pilot is documented in `PILOT.md`.
