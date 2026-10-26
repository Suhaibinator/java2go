# Milestone 116: array text and identity contracts

Array string services now return canonical UTF16 for primitive, reference and deep arrays, preserving char representation, nulls, nested cycles and caller execution for element conversion. Canonical Object array text uses the Java class descriptor and virtual hash behavior. System.identityHashCode uses allocation identity rather than virtual hashCode; primitive arguments box at the Object boundary. Legacy host APIs retain their behavior.

The coordinator independently rebuilt matching race binaries, passed runtime and compiler controls, and matched four strict JDK21/generated-race probes covering arrays, System identity, boxed arguments and comparator array text. Unchanged Comparators, FloatOrdering, MultiDimensionalArrays and NameCollisions applications pass exact JVM/Go exit/stdout/stderr parity; their generated executables were not race instrumented. Independent Sol review and separate comparator verification also pass. All original Java/expected observations are preserved.

Other hosted failures stay active. Full campaign acceptance remains round18; no full Netty compatibility claim. Durable evidence and hashes are under ignored .campaign/resume-20261002.
