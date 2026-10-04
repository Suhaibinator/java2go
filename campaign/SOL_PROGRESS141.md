# Sol checkpoint 141: owned integer text allocation

Parent: aeebed93184c25c9998d64ff45c77a1d1de1e41f (public140).

This milestone changes only `JavaStringValueOfInt` and adds two native test files. The general int32 conversion preserves Java text, raw UTF16, fresh object/backing identity, defensive copies and zero initial cached hash. Eight declared escaping values now require exactly two allocations; negative/large values previously required three, while the three sampled small values already required two. Other primitive conversions are unchanged.

Root independently completed 32 positive stages on the matching 3409-member source: 636 runtime parents under race with zero skips, seven focused parents, a separate nonrace exact-allocation gate, race all-package compilation only and pinned lint with zero issues. Two unchanged full Java applications each produced three fresh JDK outputs and three fresh generated-Go race outputs; whole stdout bytes, empty stderr and exit status match. Strict translation, all generated packages and both entries were rebuilt against this runtime. These checks do not claim the full compiler test suite ran.

The owner’s 20 stress processes retain their original attribution. A single macro Go counter pair showed approximately 0.73% fewer allocated bytes and 17% fewer mallocs; these are diagnostics, not a CPU, wall-time or JVM speed claim. Contentious native eight-fork timings, the v1 timing regression and SA4000 failure remain preserved. Original full Java/POM/oracle sources remain external and hash-pinned; this checkpoint does not publish that source corpus.

Exact-head CI140 completed with seven passing and eight failing jobs: six compiler shards, dependency applications and the aggregate test job fail. Truncated/unseen compiler diagnostics remain held; counts alone establish no absence of regression. CI141 is unknown until its actual head is checked. Private generic helper-T compilation, Arrays field-origin inference, NIO F1, the whole Reader timeout and bound/raw default remain separate held work. Full Netty and dependency acceptance remain pending.

Goal ACTIVE; latest accepted full round 18. Full CI and full-round acceptance remain false. Publication is limited to the three code leaves, `campaign-state.json` and this note. Root owns commit and push; actual postpublication identity is frozen afterward.
