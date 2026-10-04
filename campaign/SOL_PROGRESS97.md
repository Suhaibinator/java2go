# Sol progress97: source ownership across fresh contexts

Fresh superclass matchers and generic-family inventories now share declaration-to-file ownership facts. Each query still resolves names in its own lexical context. Graph replacement resets the facts, and late synthesized classes retain the ordinary lookup fallback. The change carries no semantic-result, admission or generated-body cache.

The matching public96 candidate passed160 named actions covering130 distinct tests, preserving121 existing Netty regressions,11 JVM oracles and10 metadata tests repeated in four shards. Ownership cost checks passed the original ratio limit of6. Pinned golangci-lint2.14 reported zero issues. Root independently verified the guarded eight-file delta and raw receipts. Earlier causal cost failures remain recorded.

The unchanged full1212-source Netty implementation plus eight application sources still fails strict translation at the declared300-second limit. The CLI built successfully; the partial output namespace contains ten nonempty Go files and an empty AdvancedLeakAwareCompositeByteBuf.go. These files have not passed an all-package Go build or application parity test. The timeout is not acceptance. The next diagnostic profiles this measured compiler before further repair.

This scoped milestone does not close the original Commons27, genuine Gson118, authenticated Maven dependency closure, full Netty or accumulated application/compiler acceptance gates. Checkpoint18 remains the latest accepted full round. Pending performance, reflection, collections and Gson fixes remain private until matching verification passes. No Java speed advantage is claimed.
