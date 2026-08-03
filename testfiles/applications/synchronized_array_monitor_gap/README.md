# Array synchronized-monitor parity fixture

Every non-null Java array is an object with an intrinsic monitor, so an array
reference may guard a `synchronized` block. An alias of that array denotes the
same monitor object. This fixture evaluates the aliased lock expression exactly
once, enters and exits normally, mutates through the alias inside the critical
section, then observes the mutation through the original reference. It makes
the evaluation/body/exit sequence observable as `TRACE=1234`.

This formerly failed when generated Go passed an unhashable slice directly to
the monitor registry and panicked before entering the body. The fixture now
passes and guards Java array monitor identity without relying on slice
comparability, including alias identity, exactly-once lock evaluation, normal
entry and exit, and mutation visibility through the original reference.
