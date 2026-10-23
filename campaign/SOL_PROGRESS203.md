# Sol checkpoint 203: ThreadLocal reference caching and collection inference

ThreadLocal now caches an erased Java Supplier result before a typed consumer can reject it. Failed initializers remain retryable, null results stay cached, and initializer calls retain the current Execution. The native Go callback API remains covered. Compiler adoption and SuppliedThreadLocal reflection metadata are still private work.

Native collection diamond constructors can infer their type argument from an original, verified Collection target. A checked-in application exercises seeded state across four packages, alias writes and source classes with matching simple names. Qualified member resolution also uses Cut loops to avoid temporary Split slices. One stale Affine assertion now checks the actual Go AST bindings and evaluation order.

The combined source passed all 109 qualification stages and 2,369 parent test executions. Verification includes all 881 runtime tests in normal and race modes, focused regressions, 885 F1 registry entries, full symbol/parser race tests, six build/test-compilation/lint profiles, and unchanged resolver allocation/native-call limits. Nine fresh JDK21 observations matched nine fresh race-built Go executions, including stdout, stderr, exit status and declared file effects. Existing nested Go children that run without race instrumentation receive no inferred race credit.

On the fixed private Netty query workload, allocations decreased from 360 to 192 while native calls stayed at 448 and all 32 semantic records matched. The measurement-only test is excluded from the public patch. This does not establish a full compiler or application speed improvement.

Full Supplier28, Gson112 and Netty1220 acceptance remains open. The latest full Netty attempt on checkpoint 202 timed out after 300.028 seconds with 24 file starts and 23 nonempty generated files; it never reached Go build or application execution. Hosted CI on checkpoint 202 retained the same 26 failing compiler parents as checkpoint 201, with 2,324 passing compiler parents and no new failure names. Pending source ArrayList, Gson erasure and TreeMap copy fixes remain separate adversarial work.
