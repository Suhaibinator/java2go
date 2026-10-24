# Primitive-array write experiment: recommend the unsigned predicate

A one-predicate change in an **isolated copy of the real stdjava runtime** made the complete primitive-array write path inlineable and consistently improved three focused Go benchmarks. No repository runtime/transpiler implementation was edited, and no application benchmark was run after the change. This is evidence for a candidate patch, not evidence that generated Go now beats Java.

`recommendation.patch` changes only `primitiveArrayIndex`:

```diff
- index64 := int64(index)
- if index64 < 0 || index64 >= int64(len(array.Elements)) {
+ if uint64(index) >= uint64(len(array.Elements)) {
      panic(NewArrayIndexOutOfBoundsException("array index out of bounds"))
  }
- return int(index64)
+ return int(index)
```

Keep the existing preceding null check and exception constructors/messages unchanged. The `PrimitiveArrayAssign` API and its argument evaluation order remain unchanged.

## Why this predicate preserves behavior

`javaArrayLength` admits signed int/int8/int16/int32/int64 and unsigned uint8/uint16. Array length is a nonnegative Go int. Every negative admitted index converts to an unsigned value above any possible valid slice index; nonnegative admitted indices retain their numeric value. Thus one unsigned comparison rejects exactly the same indices as the original negative-or-upper-bound check. Conversion back to int happens only after proving the index lies below the slice length, including on a 32-bit target. Nil is still checked first, and the saved target/index/RHS are still evaluated before entering `PrimitiveArrayAssign`.

The shared helper also serves `PrimitiveArrayGet`; that path needs the same regression coverage before a live implementation change. The patch deliberately does not modify the reference-array helper or introduce an unchecked public path.

## Compiler and assembly evidence, Go 1.27.1 arm64

| Variant | Index-helper inline cost | Assign-helper inline cost | Success-path call in benchmark caller |
|---|---:|---:|---|
| Baseline | 64, inlineable | 84, above budget80 | `PrimitiveArrayAssign` |
| Unsigned predicate | 57, inlineable | 77, inlineable | Neither helper |
| Separate noinline panic helpers | 140, not inlineable | 77, inlineable | `primitiveArrayIndex` |

This demonstrates why checking only whether the outer assignment wrapper inlines is insufficient. Separating panic construction made the inner helper more expensive in the inliner's cost model. The unsigned variant removes both helper calls. Compiler assembly still contains the actual word store, the explicit Java bounds branch and a Go slice bounds branch; this is **not complete bounds-check elimination**. The loop was not removed as dead work. For diagnostics, compiler `-S` output was used because this host's Go tool bundle does not expose `go tool objdump`.

## Brief measurements

Each variant imports its own full runtime copy. `diff -qr` confirmed that `stdjava/reference_arrays.go` was the only differing runtime file. Six fresh process blocks rotated baseline/unsigned/cold ordering; each process ran three benchmarks with `-test.benchtime=200ms`, GOMAXPROCS2, GOMEMLIMIT512MiB and default GOGC. Each block therefore contains one independent process per variant. Existing desktop/repair activity and no hard CPU affinity limit precision. All timed operations reported **0 B/op and 0 allocs/op**; these probes measure write/helper costs, not allocation or GC.

`RuntimeAssign` writes through the actual runtime API with dependent returned values and masked indices. `ForcedCall` adds a `//go:noinline` wrapper, exposing the cost of an extra call boundary and lost optimization context; its difference is not a universal bare CALL-instruction cost. `CanonicalFill` performs 28 actual helper writes per operation and consumes a value after each fill.

| Benchmark | Variant | Six samples, ns/op | Median ns/op |
|---|---|---|---:|
| RuntimeAssign | baseline | 1.138, 1.127, 1.193, 1.545, 1.159, 1.1 | 1.1485 |
| RuntimeAssign | unsigned | 0.479, 0.47, 0.5157, 0.545, 0.4708, 0.4253 | 0.4749 |
| RuntimeAssign | cold | 2.395, 2.151, 2.16, 2.363, 2.097, 2.011 | 2.1555 |
| ForcedCall | baseline | 1.746, 1.689, 1.694, 1.921, 1.615, 1.741 | 1.7175 |
| ForcedCall | unsigned | 1.253, 1.187, 1.401, 1.889, 1.209, 1.071 | 1.231 |
| ForcedCall | cold | 2.45, 2.223, 2.227, 2.267, 2.268, 2.186 | 2.247 |
| CanonicalFill | baseline | 85.52, 83.79, 91.75, 111, 82.68, 80.74 | 84.655 |
| CanonicalFill | unsigned | 13.76, 13.61, 14.78, 14.5, 12.4, 11.67 | 13.685 |
| CanonicalFill | cold | 71.07, 57.86, 64.42, 59.79, 61.83, 54.77 | 60.81 |

Unsigned/baseline geometric ratios from paired process blocks were 0.402 for RuntimeAssign (exploratory bootstrap95% 0.379–0.422), 0.757 for ForcedCall (0.678–0.857), and 0.151 for CanonicalFill (0.141–0.160). These intervals quantify this small experiment only. The cold variant worsened scalar writes despite improving the fill somewhat; recommend the smaller unsigned-predicate patch instead.

Do not multiply these subnanosecond savings by the application's write count to predict wall-clock savings. Register pressure, array lengths, surrounding arithmetic, exception code size, instruction cache, GC, and compiler context differ. The earlier whole-application pilot remains unchanged: Go was slower there. It takes a new parity-gated application A/B to determine whether this candidate closes any of that gap.

## Behavior checks completed

All three variants matched the same JDK21.0.6 oracle **byte for byte** for 12 cases: valid assignment/result, null receiver, null with negative index, negative/end/min/max int indices, RHS throw overriding null or bounds failure, receiver throw before index/RHS, index throw before RHS/null failure, and RHS replacement of the current array while the saved original target receives the store. Traces verify R/I/V evaluation order, exception simple type, returned value, original-array mutation, and replacement-array state.

All variants additionally passed Go helper tests for min/max int64, min int8, max uint16, and a valid uint16 index. Baseline and unsigned runtime copies each passed **all `go test ./stdjava -count=1` tests** (0.521s and 0.527s in the measured snapshot). The reproducible scripts were then exercised again on fresh isolated copies with measurements disabled; see the source for exact commands and gates.

This test set is meaningful evidence, not a claim to exhaust all Java behavior. Before adoption, run the existing generated parity/application suite, especially array assignment timing and primitive-array/null exceptions; verify int/double code generation and compiler diagnostics on supported target architectures; then reserve a quiet slot for a frozen application A/B. Do not add a compiler-text golden assertion that fails on unrelated toolchain inliner changes.

## Reproduce without editing live runtime

From the repository root, choose an empty scratch directory:

```sh
python3 performance/codegen/array_assign_probe/prepare.py /tmp/java2go-array-check
python3 performance/codegen/array_assign_probe/run.py /tmp/java2go-array-check
```

The first script snapshots stdjava and creates baseline/unsigned/cold copies. The second compiles all variants, saves inline/BCE diagnostics and assembly, compiles the Java oracle, checks every variant against it, runs wide-index tests and baseline/unsigned runtime tests. It performs **no timing by default**. After reserving a short measurement window:

```sh
python3 performance/codegen/array_assign_probe/run.py /tmp/java2go-array-check --measure
```

This writes exact benchmark output per block plus `measurements.json` in scratch. Original experiment diagnostics/raws are at `/tmp/java2go-array-assign-probe/`; the complete timing samples are preserved in the compact table above. No sustained application suite, runtime tuning or production patch application is hidden in these scripts.

The recommendation is stored as a zero-context patch; apply with `git apply --unidiff-zero recommendation.patch` only when implementation and full validation are scheduled. It is not applied to the live runtime.
