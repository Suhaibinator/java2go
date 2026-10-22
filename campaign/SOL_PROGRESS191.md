# Sol checkpoint 191: source generic demand syntax routes

The compiler now retains syntax routes for source generic demand discovery within one active compilation. Each query resolves those routes against current symbols, methods, imports, headers and binders. Source bytes, parse roots, graph ownership and session lifetime are checked before replay. Unknown or changed state uses the complete existing scanner; closing the invocation retires the inventory. Resolved demand results are not cached.

The normal work-growth regression measures 32, 64 and 128 units for 4, 8 and 16 files, compared with the prior 1,600, 3,200 and 6,400. A sixteen-query calibration reduces Cgo calls from 20,960 to 13,072 and allocations from 9,392 to 8,496. These are compiler measurements; full Netty conversion time and generated application speed remain unmeasured.

Independent verification rebuilt the exact current-checkpoint composition and completed all 69 stages without failures, skips or timeouts. Its 1,285 parent executions include 348 compiler passes, 63 UTF8 checks, all 841 default runtime race tests, 23 symbol tests and 10 parser tests. All six default/F1 build, test-compilation and lint checks pass. Whole standalone and Maven conversion outputs remain equal. The coordinator rehashed every source file, delivery pin, raw receipt and stream and confirmed sequential execution, process cleanup and bounded stages.

Hosted checkpoint 190 CI failed overall: seven jobs succeeded and eight failed. All 47 focused parents and application parity passed. The same 37 compiler parent failures and eight skip events remain, with no additional failure or skip names. All fifteen job logs and 120 sealed CI artifacts were independently checked. Full CI and round acceptance remain open.

The Netty cache application remains an active acceptance test after its strict conversion and generated race builds passed. Three array-bounds messages and one null-receiver message still differ from the JVM oracle. Their repairs, generic source selection, collector body erasure, deferred stream interference and native startup services remain private pending their own verification. The Sol campaign continues.
