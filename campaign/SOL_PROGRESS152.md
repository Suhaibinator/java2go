# Sol checkpoint 152: array method strict diagnostics

Calls to nonexistent methods on statically proved array receivers now reach the existing unsupported-expression diagnostic. For example, `Thread[].interrupt()` no longer escapes strict mode as an unresolved Go selector. Array rank comes from the receiver declaration with assignment target typing cleared. Unknown receivers keep their existing route, and Object/clone method names retain existing signature and access handling.

The coordinator independently reproduced the unchanged original test failing on public151, then passing with the repair. Fresh JDK21 compilation rejects the invalid Java; legal array cloning and Object methods remain valid. Fourteen focused and existing compiler test parents, all672 runtime parents, default/f1 all-package race builds and test compilation, and golangci-lint2.15 in both profiles pass. No assertion or original fixture was changed.

This closes one known compiler regression, not the full replay. The historical private826 replay stopped at this failure after707 focused passes; a coherent successor must rerun its affected acceptance gates. A separate private cast/string component join passed42 Go comparisons and744 runtime tests, but it is not included in this publication. Class.cast, FileChannel source construction, Throwable stack contracts, full dependency-source closure, complete CI/round acceptance and full Netty parity remain open. Last accepted full checkpoint remains18; no performance advantage is claimed.

Raw verification artifacts remain outside tracked source under `root-public152/run/ROOT_RAW_AUDIT.json`; the active resume pointer is recorded in campaign-state.json.
