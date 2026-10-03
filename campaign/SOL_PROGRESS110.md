# Milestone 110: String.contains contracts

String.contains now searches canonical UTF16 units and invokes the CharSequence virtual toString once using caller execution. Null checks, isolated surrogates, empty text, exception identity, argument ordering, thread identity and held monitors have focused coverage.

The coordinator independently rebuilt matching race test binaries, passed two runtime tests with three null subtests and one compiler shape test, and verified two strict JDK21 applications with all generated packages and entry points built with race instrumentation. Unchanged Strings and ObjectStrings applications also pass byte-for-byte JVM/Go exit, stdout and stderr comparison; their generated executables were not race instrumented. Sol retained fresh red evidence and independent source review.

Other compiler, dependency and application CI failures remain active. Full campaign acceptance remains round18. Durable raw evidence and source hashes are in ignored .campaign/resume-20261002.
