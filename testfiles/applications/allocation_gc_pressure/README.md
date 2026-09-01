# Allocation and garbage-collection pressure fixture

This deterministic application is both a byte-for-byte parity fixture and a
CPU/memory benchmark payload. It creates memory pressure through ordinary Java
allocation patterns rather than runtime-specific GC controls or management
APIs, so the same source can be transpiled and compared fairly.

## Workload

- Allocate batches of short-lived `ScratchRecord` objects, each owning two
  independently allocated integer arrays. A batch remains reachable until all
  records are consumed, then the following batch replaces the entire cohort.
- Build retained `Cohort` graphs containing nodes with payload arrays and both
  `next` and `skip` links. The links form cycles and non-local edges rather than
  a flat collection of unrelated objects.
- Traverse and mutate every live cohort after each phase, ensuring the retained
  graph is semantically observable and cannot be discarded as dead allocation.
- Rotate six retained slots over 3,456 phases. Replacing a slot releases an older
  cyclic graph while newer generations remain live, exercising mixed object
  lifetimes and repeated collector reclamation.
- Traverse every retained node and payload at the end, reducing all live state
  into order-sensitive integer checksums.

The program prints workload dimensions, lifetime invariants, and stable
checksums only. It never reads clocks, random sources, files, stdin, the
environment, the network, or implementation-specific garbage-collector data.
There is no explicit `System.gc()` call and no timing output.

At the configured scale, 35,389,440 short-lived records pass through the
bounded batch window while only six cyclic cohorts remain retained. Three
calibration runs on an Apple M4 Max took 11.92–12.28 seconds for Java and
11.40–11.61 seconds for generated Go. Observed peak RSS stayed below 1.28 GiB
for Java and 15 MiB for generated Go. These measurements document workload
scale; they are not machine-specific pass/fail thresholds.

`expected.stdout` is the exact Java 21 oracle. The application parity harness
also requires the generated Go program to compile and match it byte for byte.
The external benchmark harness compiles first, runs an untimed validation pass,
then measures one fresh Java or generated-Go process per benchmark sample. The
long-running payload makes each `-count` sample an independent process-level
measurement without multiplying a single benchmark operation unnecessarily.
