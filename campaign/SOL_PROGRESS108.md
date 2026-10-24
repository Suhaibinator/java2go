# Milestone 108: collector merge callback execution

Collectors.toMap merging now unwraps the canonical BiFunction adapter with the caller Execution and its resolved value type. The merge expression is captured once. The application workflow and runtime collection algorithm are unchanged.

Fresh TDD reproduced the original Go compilation failure and then passed the JDK-derived merge result, thread identity and held-monitor checks. The coordinator independently passed all eight existing collector roots and the new semantic regression with race instrumentation, plus unchanged Collectors_ and java_collection_contracts JVM/Go fixtures in strict mode. These two original e2e programs use their existing nonrace generated builds; the focused generated tests were race instrumented. Expanded verification found one further stale reduce callback assertion; both typed signatures now include Execution, preserving all eight embedded Java programs and checks.

Raw evidence, including the initial stale assertion failure, source hashes, shim invocations and independent source review, is durable under ignored .campaign/resume-20261002. No failures or skips were waived. Full CI and the frozen dependency gates remain open; round18 remains the latest full campaign acceptance.
