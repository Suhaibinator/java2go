# Array synchronized-monitor known gap

Every non-null Java array is an object with an intrinsic monitor, so an array
reference may guard a `synchronized` block. An alias of that array denotes the
same monitor object. This fixture evaluates the aliased lock expression exactly
once, enters and exits normally, mutates through the alias inside the critical
section, then observes the mutation through the original reference. It makes
the evaluation/body/exit sequence observable as `TRACE=1234`.

Generated Go represents the `int[]` as an `[]int32` slice and currently passes
that slice directly to the monitor registry. The registry uses interface values
as map keys, but a Go slice is not hashable, so monitor entry panics before the
body with `panic: hash of unhashable type: []int32`. The fixture pins
that earliest stable runtime failure until monitor identity supports Java array
objects without relying on slice comparability.
