# Milestone 112: primitive stream array overloads

IntStream.of, LongStream.of and DoubleStream.of now select the array overload for a single matching primitive array or null. Scalar, empty and expanded forms retain their existing handler. Two old String component shape assertions were migrated to canonical references with their Java sources unchanged.

The coordinator independently rebuilt matching race binaries, passed ten focused and adjacent compiler regressions, and matched two strict JDK21 applications with all generated packages and entry points race built. They exercise all three families, nulls, argument evaluation once, empty arrays, scalar forms and same-named source classes/binders. Unchanged NumericStreams also passes exact JVM/Go parity; its generated executable was not race instrumented.

Array-backed streams still copy eagerly, so lazy mutation behavior remains an explicit compatibility gap. Other hosted failures stay active; full campaign acceptance remains round18. Evidence and hashes are retained under ignored .campaign/resume-20261002.
