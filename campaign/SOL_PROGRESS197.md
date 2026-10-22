# Sol checkpoint 197: deferred stream runtime core

The runtime now includes deferred stream sources, per-evaluation pipeline state, terminal claim checks, current execution propagation, cancellation, sorted barriers and cleanup composition. List.Clear zeros vacated backing slots so stream interference observes Java-style cleared references. Existing compiler output and the legacy nil-pipeline API are unchanged.

All 31 validation stages passed, with 1,482 test-parent executions and no skips, timeouts or payload retries. The 27 new portable parents passed once normally and twenty times with race instrumentation. Full runtime (876 parents), symbol (23) and parser (10) race suites passed. Default/F1 whole-project race build, test compilation and lint passed, as did five 4096-byte partial-write checks and uncapped lease-close publication. The coordinator and an independent Sol reviewer verified sources, raw receipts, environments and process cleanup.

This publishes the runtime prerequisite, not Java adoption. Exact sorted callback traces and Clear+Add capacity behavior remain open. The original Supplier28 application now passes strict translation and both Go race builds; its first execution fails at a source-defined supplier viewed as Supplier<?>. That original acceptance test remains active, along with Gson112 and Netty1220.
