# Sol checkpoint 149 — FileChannel runtime protocol and NIO metadata

Adds a canonical runtime view for compiler-owned source FileChannel allocations, with declaration-selected read callbacks, shared Java identity, and reentrant close handling. Native channels continue through the existing platform implementation. Signature-only channel, buffer, lock and map-mode types do not implement the corresponding Java services.

Fresh JDK 21 observations exposed incorrect interface superclasses and modifier flags in the new declaration registrations. The fix registers interface kind and declared modifiers for all 16 types, including the non-final public static MapMode class. The unchanged differential test fails before the fix and passes afterward.

Independent root verification used Go 1.27.3 and JDK 21.0.6: 671 runtime parents and 17 symbol parents pass with race instrumentation; six focused parents pass three times; 20 fresh processes exercise close concurrency; 24 JVM observations match the runtime tests’ reference results. Default and f1_os_profile all-package race builds, test compilation and golangci-lint 2.14.0 pass. The public148 baseline’s 665 runtime parents also pass freshly with Go 1.27.3. All 75 candidate verification stages exit zero without timeout or cleanup failure.

This is a runtime prerequisite. Compiler admission, source-subclass lowering and current-Execution routing are still pending. SourceCarrierMain has JVM-only evidence in this gate. Native dynamic class/interface/array membership, the full tagged runtime, and other platforms are not established here. Full CI and the accumulated application campaign are not accepted by this checkpoint; the last accepted full round remains 18. No full Netty compatibility or performance advantage is claimed.

The parallel exception repair matches its seven retained applications, but a new valid dependency application and reducer reveal a suppression-storage collision with a legal Java business method. That failure remains active. Reader declaration-origin repair and private compiler-union integration also continue separately.

Raw evidence remains outside tracked source under `.campaign/resume-20261002/resume-20261020/root/stageA-metadata-independent/`. `ROOT_VERIFIED.json` records the source and receipt-index hashes. The campaign remains active with Sol agents only.
