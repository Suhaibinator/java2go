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
- Rotate six retained slots over 72 phases. Replacing a slot releases an older
  cyclic graph while newer generations remain live, exercising mixed object
  lifetimes and repeated collector reclamation.
- Traverse every retained node and payload at the end, reducing all live state
  into order-sensitive integer checksums.

The program prints workload dimensions, lifetime invariants, and stable
checksums only. It never reads clocks, random sources, files, stdin, the
environment, the network, or implementation-specific garbage-collector data.
There is no explicit `System.gc()` call and no timing output.

`expected.stdout` is the exact Java 21 oracle. The application parity harness
also requires the generated Go program to compile and match it byte for byte.
The external benchmark harness compiles first, runs an untimed validation pass,
then measures three fresh Java or generated-Go processes per benchmark sample.
