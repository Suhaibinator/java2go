# Integer and branch search benchmark

This deterministic application combines four sustained CPU-focused workloads:

- twenty rounds of exhaustive bit-mask N-queens searches over four
  row-constraint variants,
  recording explored nodes, branches, dead ends, solutions, and order-sensitive
  solution signatures;
- twelve rounds of an integer-only Eratosthenes sieve with prime counts, sums,
  and an overflow-sensitive signature;
- sixteen deterministic fills and sorts of 262,144-element integer arrays,
  followed by 1,048,576 total hit/miss binary-search probes and a comparison
  count; and
- eighteen dependent pointer-chase rounds through two 1,048,576-element integer
  tables with deliberately uneven branches, mixed overflow arithmetic, bit
  operations, and irregular memory access.

Runtime is scaled through repeated bounded computations rather than sleeps,
timing loops, giant arrays, or expanding search frontiers. A round's largest
logical allocation is the walker's pair of integer tables (about 8 MiB total),
and rounds execute sequentially so prior tables are reclaimable. This leaves a
wide safety margin below the 8 GiB ceiling while still sustaining roughly
11–14 seconds of Java CPU work on the calibration Apple M4 Max.
Final manual Java samples took 11.71–11.74 seconds, while the generated Go
sample took 12.07 seconds. The repository benchmark measured 14.349 seconds per
Java run and 13.203 seconds per generated-Go run. Peak RSS was 143.1 MiB for
Java and 20.6 MiB for generated Go. These figures document the reference host;
they are not portable pass/fail thresholds.

The checked-in output contains semantic totals and independent checksums for
each workload. There are no clocks, random sources, files, environment inputs,
network calls, or concurrency, so Java and generated Go must produce identical
bytes on every run. `benchmark.json` opts the fixture into the application
performance runner with one fresh Java process and one fresh generated-Go
process per measured benchmark operation. Use benchmark repetition (`-count`)
when multiple independent timing samples are desired.
