# Sol checkpoint 198: import lookup allocation and identity assertions

Source-type lookup borrows immutable cached static-import tuples while the public import API continues to return caller-owned copies. Namespace, accessibility, lexical precedence, source/tree identity and lifetime checks remain active. In the matched 64/512-import workloads, allocated bytes per query fell from 2,688/21,760 to zero; cgo calls stayed at five. This measures lookup allocation, not application or Java-relative speed.

Two compiler regression assertions now verify complete enum-reference and boxed-Class identity expressions through AST structure and 31 negative controls. Their original Java, behavioral expectations and existing Gap Go companion remain unchanged. Independent JVM/Go runs and generated Go witnesses passed. Nested child tests were explicitly run in normal and race modes.

Combined qualification passed 90 accepted gates and 1,451 test-parent executions, including full runtime (876), symbol (23) and parser (10) race suites and default/F1 race builds, test compilation and lint. An earlier twenty-repetition race batch hit its 55-second native deadline after seven repetitions. That failure remains preserved with no credited passes; twenty separate invocations completed under unchanged 55/65-second limits. Sources and tool pins stayed unchanged. The coordinator independently rechecked receipts, raw streams, execution modes and owned process cleanup.

Hosted checkpoint 197 retains the same 30 compiler failures as 195 and all 15 previously repaired tests pass. Hosted 198, the known Gson adversarial failure and the original Supplier28/Gson112/Netty1220 acceptance gates remain open.
