# Sol checkpoint 171: retain proved generic widening costs during overload selection

A generic Collection<E> overload could lose to Object even when declaration-owned inference proved the collection conversion had a lower widening cost. The scorer previously adopted only exact projected conversions. It now also adopts an applicable, proved projection with a lower cost, retaining existing invocation phases and bound checks. Caller and callee binders remain distinct.

The coordinator independently reconstructed and reversed the one-condition change and two new test leaves against public170. The unchanged full static-import workflow passes three JVM/Go comparisons with output 1:1. Three owned regression parents pass twenty race repetitions each. Of 66 related parents, 65 pass; the existing typed-varargs rendering assertion fails identically on current baseline and candidate, with unchanged assertions and matching diagnostics. No pass credit is assigned to that failure.

Thirty-nine preceding Gson parents, both Reader workflows (six JVM and six Go executions), both original exception CI parents, full runtime801/symbol23/parser10 race registries, and default/F1 race builds, test compilation and lint pass. All90 process receipts were independently rehashed: 88 positive plus the retained varargs failure on baseline and candidate. A verifier success-label erratum is preserved; completed payloads were not retried.

Evidence: root-score171/run/ROOT_RAW_AUDIT.json. Full CI, full Netty, the accumulated round and performance gains remain unaccepted. The five-minute Netty timeout remains active.
