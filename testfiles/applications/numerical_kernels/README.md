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
Three calibration runs took 11.47–11.72 seconds for Java and 16.76–18.81 seconds
for generated Go. Observed peak RSS was 1,249.3 MiB for Java and 30.6 MiB for
generated Go, both comfortably below 8 GiB. These measurements describe the
reference host rather than imposing machine-specific test thresholds.

`expected.stdout` was captured from the Java 21 oracle. The application parity
harness requires the generated Go program to compile and match it byte for byte.
