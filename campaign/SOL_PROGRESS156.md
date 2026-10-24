# Sol checkpoint 156: FileChannel write runtime

The runtime now provides native relative FileChannel writes and source-first virtual facades for relative and positional writes. Source callbacks receive the caller execution and own validation and effects. Legacy read-only source protocols remain valid; a partially supplied write pair is rejected. Native relative writes use the shared descriptor cursor and release the lifecycle mutex before acquiring a descriptor lease. The buffer window is copied for signed-byte conversion.

Independent verification passed all678 runtime and17 symbol tests with race detection, source/native/lease checks repeated20 times, three bounded partial-write runs, the existing positional partial-write profiles for seeds17/41/97, both build profiles and golangci-lint2.15 with zero issues. FreeBSD passed compilation only. The published155 runtime baseline passed672 tests; its missing new API was recorded as a compile gap. Frozen JDK21 reference observations remain unchanged.

This checkpoint changes the runtime and adds regression tests. Compiler lowering and the full Netty application remain separate integration work. Active cancellation during a native write, broader platform behavior, full accumulated CI/round acceptance and Netty JVM/Go parity remain open. No generated-program performance gain is claimed. Last accepted full checkpoint remains18; the Sol-only campaign continues.

Verbose evidence stays outside tracked source under `root-public156/run/ROOT_RAW_AUDIT.json`.
