# Numerical kernels parity and benchmark fixture

This deterministic application exercises numerical code shapes that are useful
for comparing Java with generated Go. It performs a fixed amount of work and
prints no timings, so the same program is both a byte-for-byte parity fixture
and a reusable benchmark payload for an external timing harness.

## Workload

- Construct two dense matrices from an integer recurrence with no random API.
- Run sixteen cache-sensitive blocked matrix products over flat row-major storage.
- Run an iterative five-point stencil that allocates a fresh matrix on every
  pass and applies a bounded polynomial transform.
- Run a separate floating-point vector recurrence with dependent arithmetic and
  repeated allocation.
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

`expected.stdout` was captured from the Java 21 oracle. The application parity
harness requires the generated Go program to compile and match it byte for byte.
