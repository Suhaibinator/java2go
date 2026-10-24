# Sol checkpoint 192: verify enum behavior by execution

One CI test still required the generated text `return dy.enumName`, although enum conversions now use the JavaString implementation. The same test also required an obsolete String.valueOf call chain, hidden behind the first assertion. Its failure was reproduced independently on the unchanged public checkpoint and the private Netty candidate.

The test now executes the generated enum and its original printAll method. It checks the compiled fmt.Stringer interface, both exact UTF16 names, reference-preserving default conversion, original stdout for printing/concatenation/String.valueOf, null text conversion, and genuine Java NullPointerException values. The original Java declarations, every sibling test, and all compiler/runtime code remain unchanged. The historical failure is preserved.

Three fresh JDK21 runs match three strictly transpiled Go runs in complete stdout, stderr and output files. The emitted behavior passed twenty normal and twenty race repetitions. Related enum controls and all six default/F1 build, race-test-compilation and lint checks pass. The coordinator audited all 42 raw receipts and streams: 41 succeeded, with one retained initial test-helper name-matcher premise failure. There were 86 successful native parent executions. No new skip or assertion waiver was introduced.

Hosted checkpoint 191 CI failed overall with the same 37 compiler parent failures and eight skip events. All 54 focused parents and application parity passed. All 121 sealed CI artifacts were rehashed. This checkpoint repairs one baseline test; a new hosted result is still required before reporting its CI effect.

The unchanged five-source cache reducer and array control both achieved fresh JVM18/Go18 parity privately. They exercise prerequisites of the actual Netty compiler frontier; they are not a full Netty application. New lint findings in that candidate still block its integration and the next full 1220-source gate. Supplier28 currently has three generated-Go compilation errors. Full Netty, Gson, CI and round acceptance remain open. The Sol campaign continues.
