# Milestone 111: integral stream statistics

Translated IntSummaryStatistics and LongSummaryStatistics now accumulate sums with Java long width. Integral averages use that sum before conversion to double, preserving cancellation above 2^53 and long overflow. Native Go callers retain the existing element-width GetSum API; translated integral getSum uses GetSumLong.

The coordinator independently rebuilt matching race binaries, passed nine runtime regressions and a compiler shape check, and matched a strict JDK21 statistics application with generated Go packages and entry point built with race instrumentation. The unchanged NumericStreams application also passes exact exit/stdout/stderr parity; its generated executable was not race instrumented. Sol retained fresh red evidence; independent review approved the scoped changes.

Other hosted compiler, application and dependency failures remain active. Full campaign acceptance remains round18. Raw evidence and source hashes are retained under ignored .campaign/resume-20261002.
