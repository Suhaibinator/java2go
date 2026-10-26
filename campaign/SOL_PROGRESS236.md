# Filesystem service checkpoint 236

Files.createDirectories now returns its input Path and follows the exercised JDK mkdir-first and ancestor fallback contracts. The repair preserves exception classes, UTF16 messages, causes, partial filesystem effects and cleanup. An isolated child verifies that an empty Path does not recreate its removed working directory. These are Go implementations of JDK/platform services; dependency implementation sources remain translated.

All 57 actual qualification payloads passed and were independently audited. The existing 939 runtime parents and 17 JDK-derived filesystem controls passed in normal and race modes. Fresh Commons Codec conversion preserved all generated implementation bytes and matched complete declared JVM/Go observations across 24 semantic child runs, 54 workflows and 162 reports. Application execution used normal binaries; race binaries were built and their provenance checked. Native runtime suites executed with race instrumentation.

The filesystem tests live under stdjava/filesystemcontract, so the existing runtime CI command discovers them. Lint returned zero issues. Its initial controller expected empty stdout and stopped on the valid `0 issues.` line; the original attempt remains preserved, and an exact decoder adopted the successful result without repeating payloads.

Native semantics were verified on Darwin. Linux source was reviewed and the new tests will run in hosted CI; other GOOS retain an explicitly approximate access/error path. No timing, allocation gain or generated-program speedup is claimed. The separate private concat optimizer is excluded.

Public235 CI passed all 2631 compiler parents, 4570 compiler children, 939 runtime parents and 1026 unit parents without failures or skips in those suites. One prior URL refusal child moved to positive conformance with identical Java text. Gson advanced from the URL conversion refusal to generated Go compilation, where generic, collection and reflection gaps remain. New-head CI, the active application campaign and Netty integration remain required; this checkpoint does not accept a campaign round.
