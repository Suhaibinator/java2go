# Milestone 104: portable JDK selection for CI tests

The campaign String oracle used an absolute macOS JDK path, which does not exist on Ubuntu CI. It now uses the existing `campaignCompilerJavaTool` resolver to select `java` and `javac` through `JAVA_HOME` or `PATH`. Missing tools remain fatal.

Only the test helper changed. All Java sources, Go drivers, process arguments, expected output and assertions remain unchanged. A fresh race-enabled native test build and both original affected tests passed with zero failures or skips and clean process cleanup. Independent source and actual-execution reviews passed; the coordinator separately rehashed all 3,111 source members, both command receipts, four raw streams and the generated test binary.

The earlier Ubuntu failure on `b3b88f79` is retained as the causal evidence; its helper preimage is byte-identical to the public103 helper. These local checks use explicitly selected JDK 21 and do not constitute a full Ubuntu CI replay.

Full CI and the campaign round remain unaccepted. Larger compiler/runtime and dependency application failures remain active. Round18 is the latest accepted full campaign round.
