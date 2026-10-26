# Sol checkpoint 151: canonical native Object allocation

`new Object()` now carries `java.lang.Object` identity for getClass and nominal checks. Its nonzero-sized pointer remains the shared identity for equality, hashing and monitors; unrelated host pointers remain unknown.

The coordinator rebased one allocation return and two new files onto public150 (`81a1920`), without importing the private compiler union. A compiled baseline regression fails on missing metadata; the candidate passes all four subcases. Fresh race runs pass all671 existing runtime parents and all672 candidate parents, plus20 fresh object/monitor processes. Three JDK21 observations are stable. Default and f1 all-package race builds/test compilation and golangci-lint2.15 pass.

This is a scoped runtime milestone. The full private compiler replay remains RED:707 focused tests passed before the known invalid-array strict-diagnostic failure708. Remaining stages are unverified. Full cast C7, Class.cast, source FileChannel construction, dependency-source closure, full Netty parity and complete CI/round acceptance remain open. Last accepted full round remains18; no performance advantage is claimed.

Detailed evidence remains outside tracked source: `root-public151/run/ROOT_RAW_AUDIT.json`, with source and tool hashes. The active resume pointer is recorded in campaign-state.json.
