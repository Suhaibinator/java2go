# AbstractList platform checkpoint 239

The exercised AbstractList/RandomAccess service now supports translated source subclasses through their own get, size and removeRange implementations. The repair preserves tested reference casts, null and bounds behavior, iterator invalidation, virtual dispatch and partial array-copy effects. The real Gson NonNullElementWrapperList implementation remains unchanged Java source translated to Go.

All 94 component stages and 20 publication closure stages passed and were independently audited. Each normal/race mode passed 174 compiler parents and 161 subtests, 946 runtime parents and 3030 subtests, and 15 parser parents and 84 subtests. Fresh real-wrapper JVM and generated race-Go executions matched the complete 347-byte stdout; generated implementation bytes remained intact. All newly added ordinary test leaves executed in both modes.

The actual public238 inventory of 2662 compiler parents remains present; the candidate has 2679 parents. All 17 original superclass-query budget parents passed with atomic coverage and race-plus-atomic coverage, preserving their original assertions, shields, counters and limits. Current public lint reports zero issues. This is inventory preservation and focused execution, not a claim that all 2679 compiler parents ran locally.

The 47-path regional transport preserves current238 Enum, Collection, interface-storage, Date, NIO and covered-test changes. The separately discovered anonymous captured-type proof issue remains open and is not part of this checkpoint. No universal List/ArrayList ABI, full Gson112, Netty or campaign-round acceptance is claimed. New-head hosted CI remains required.
