# Integer and branch search benchmark

This deterministic application combines three CPU-focused workloads:

- exhaustive bit-mask N-queens searches over four row-constraint variants,
  recording explored nodes, branches, dead ends, solutions, and order-sensitive
  solution signatures;
- an integer-only Eratosthenes sieve with prime counts, sums, and an
  overflow-sensitive signature; and
- a dependent pointer chase through two large integer tables with deliberately
  uneven branches, mixed integer arithmetic, bit operations, and irregular
  memory access.

The checked-in output contains semantic totals and independent checksums for
each workload. There are no clocks, random sources, files, environment inputs,
network calls, or concurrency, so Java and generated Go must produce identical
bytes on every run. `benchmark.json` opts the fixture into the application
performance runner with three fresh Java processes and three fresh generated-Go
processes per measured benchmark operation.
