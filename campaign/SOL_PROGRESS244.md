# Generic source-class storage closure

Generic source classes can participate in the same fully audited storage-component closure as source interfaces. The planner chooses a source class anchor when no generic source-interface anchor exists, then performs a fresh, monotone audit before activating its representation. Unsupported bounds, foreign declarations and incompatible storage still refuse lowering.

Anonymous classes also finish receiver canonicalization for their exact synthetic owner after their methods are emitted. Named, nested and constructor/function binders retain their own identities.

The exercised cases include a generic Date adapter, mutable Date identity, Timestamp nanoseconds, nested source classes, cyclic component dependencies and delayed cast failures. Original Java inputs and negative controls remain unchanged.

Fresh qualification on the public243 base passed all 32 stages. Each normal and race mode passed 60 distinct compiler parents and 214 subtests; both binaries registered exactly 2,719 public tests. Four JVM/Go helper observations per mode matched, generated packages and entries built with race instrumentation, lint reported zero issues, and all owned processes closed. Independent actual audit and coordinator receipt/stream checks passed. This is focused execution, not a full 2,719-test run.

The broader 236-stage qualification, runtime and original4059 evidence retain their earlier source scopes. This milestone does not establish full Gson or Netty compatibility.

Campaign operations: the user has authorized ongoing project-local Go build-cache cleanup. Clean at idle batch boundaries and before space exhaustion; retain sources, dependency/module and fuzz caches, logs, tools and explicit baseline/comparison binaries. Never clean while campaign builds use the cache.
