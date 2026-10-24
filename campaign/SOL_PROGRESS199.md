# Sol checkpoint 199: declaration candidate lookup

Standalone compiler invocations now index ordered, direct declaration candidates after resolution. Owner selection, ancestry, imports, accessibility, binders and warnings continue through their existing logic. Public mutable rendering retains the original lookup path. Session, render, graph, source and parser-root checks retire stale indexes; an unknown region uses the original complete search.

The matched 50/200/800-file workloads reduced allocations from 395,531/402,435/430,042 to 309,897/317,863/349,685 per batch, an improvement of roughly 19–22%. Cgo call counts stayed unchanged. These measurements cover compiler lookup allocation and do not establish generated-program speed relative to Java.

Qualification passed all 61 gates and 1,321 test-parent executions. This includes the full runtime (876), symbol (23) and parser (10) race suites, default/F1 builds, test compilation and lint, and a fresh normal compiler binary. An independent source reviewer verified every regional change and its inverse; the coordinator separately verified all raw receipts, outputs, process cleanup, input hashes and execution modes.

Hosted checkpoint 198 has 28 remaining compiler test failures, down from 30 at checkpoint 197. Both identity-expression assertions repaired in checkpoint 198 now pass, with no new failed names. The original Supplier, Gson and Netty applications remain active acceptance gates.
