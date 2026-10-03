# Netty source preparation

Netty 4.1.138.Final is pinned as an artifact proposal. The active application locks and dependency stage remain unchanged. Root rehashed all 31 captured artifacts and checked their published checksum declarations. Independent review verifies 1,212 matching Java implementation sources, including genuine shaded JCTools and template-derived generated sources.

The proposal records eight mandatory Netty modules and JCTools 4.0.5. Captured artifacts and provenance are preserved in ignored `.campaign/netty/4.1.138.Final`; the public JSON supplies exact URLs, sizes and checksums. Four artifacts lacked published SHA256 sidecars: their local SHA256 and verified published SHA1 are distinguished explicitly.

Source-build closure remains open. JDK21 activates the parent java21 profile, selecting optional JBoss Marshalling 2.0.5.Final rather than the base-profile 1.4.11.Final. Matching package-info compilation/shading, optional API providers, resources and native contracts require validation. No Netty JVM or Go execution has passed, and this checkpoint does not advance the campaign stage.
