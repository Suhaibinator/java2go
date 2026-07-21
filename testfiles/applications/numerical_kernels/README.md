# Numerical kernels parity and benchmark fixture

This deterministic application exercises numerical code shapes that are useful
for comparing Java with generated Go. It performs a fixed amount of work and
prints no timings, so the same program is both a byte-for-byte parity fixture
and a reusable benchmark payload for an external timing harness.

## Workload

- Construct two dense matrices from an integer recurrence with no random API.
- Run 1,200 cache-sensitive blocked matrix products over flat row-major storage.
- Run 160 independently checksummed batches of a 300-iteration five-point
  stencil that allocates a fresh matrix on every pass and applies a bounded
  polynomial transform.
- Run a separate floating-point vector recurrence with dependent arithmetic and
  repeated allocation for 3,400 passes over 262,144 elements.
- Consume every result with order-sensitive, quantized `long` checksums.

The matrix and vector dimensions, block size, and iteration counts are printed
as concise integer metadata. All floating-point state is reduced to stable
integer checksums; no locale- or implementation-specific floating formatting is
part of the oracle.

## Determinism and benchmark use

The application has no stdin, files, clocks, environment reads, network access,
external libraries, or calls to `Random`. Seed construction uses a fixed modular
integer recurrence. Numerical kernels use only explicit IEEE 754 arithmetic in
fixed loop orders. Matrix results retain twelve decimal places in their
checksums; the vector checksum retains six so permitted last-bit contraction
differences cannot make otherwise equivalent implementations fail parity.

For performance comparisons, compile each implementation first and time only
the launched Java class or generated Go binary. Run enough process repetitions
for the host, pin comparable CPU resources when possible, and treat the printed
checksums as the validity guard for every timed sample.

The repeat counts intentionally make one Java execution take more than ten
seconds on the Apple M4 Max calibration host. The live matrices and vectors
remain small; runtime is produced by repeated useful computation rather than by
inflating the data set or sleeping. `benchmark.json` therefore uses one process
per measured operation so every `ns/run` value represents one complete payload.

## July 2026 optimization calibration

The original affine-loop lowering cached matrix storage but still recomputed
`row * stride + column` and retained a bounds check at each accessor. Profiling
showed that generated Go executed substantially more instructions than Java at
nearly the same instructions per cycle; garbage collection was not the cause.
The transpiler now versions eligible loops once for null state and, for proven
bounded canonical column loops, proves the complete Java `int` index interval
before iterating over equal-length row slices. Overflow, invalid bounds, null
storage, short spans, and noncanonical control flow retain the original checked
path, so the optimization does not weaken Java exception behavior.

On the Apple M4 Max reference host with Java 23.0.2 and Go 1.26.5, commit
`36ffd4c` produced these fresh end-to-end results. Every warm-up and measured
process passed the byte-exact parity oracle.

| Implementation | Three samples | Mean |
| --- | ---: | ---: |
| Java | 11.242–11.386 s | 11.320 s |
| Generated Go | 11.833–12.904 s | 12.474 s |

Generated Go therefore remained 10.2% slower than Java on this sample set
(`1.102x` Java time). At the preceding null-versioning checkpoint, the same
harness measured 11.054 seconds for Java and 15.781 seconds for generated Go,
or a 42.8% gap. A same-generated-code component A/B, with only the new proof
condition forced false for the control, measured the matrix kernel 36.4% faster
and the stencil kernel 18.5% faster with row slicing enabled. Compiler bounds
diagnostics confirmed that specialized matrix and stencil accesses have no
per-element bounds checks; only one-time slice checks and the stencil's dynamic
west/east accesses remain checked.

During profiling, observed peak RSS was about 1,249 MiB for Java and 24–31 MiB
for generated Go. Both are comfortably below 8 GiB. These measurements describe
the reference host rather than imposing machine-specific performance or memory
thresholds.

`expected.stdout` was captured from the Java 21 oracle. The application parity
harness requires the generated Go program to compile and match it byte for byte.
