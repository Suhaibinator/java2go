# Sol checkpoint 193: Netty compiler prerequisites

Generic-array overload selection and erased-array result projection now preserve the exercised declaration and storage contracts. Enum switches support the tested null/default forms while keeping selector evaluation, ownership and storage proofs. The evidence now lives in a typed, invocation-local structure instead of deprecated Go AST metadata. Array-bound and helpful null-invocation messages match the exercised JDK21 cases.

The unchanged five-source cache reducer and array-control reducer pass fresh JVM18/Go18 comparisons of status, stdout, stderr, files and state journals. Generated implementations stayed byte-exact when the typed enum evidence replaced the old certificate. These reducers exercise the actual Netty frontier; they do not establish full Netty support.

All 221 finite checks passed, including 2409 native parent executions, the full 849-parent runtime race suite, symbol/parser checks and all six default/F1 build, race-compilation and lint gates. The coordinator rehashed every source file, receipt and raw stream and checked process cleanup. Independent review verified the typed evidence and regional exteriors. Original challenges and assertions remain unchanged. Two infrastructure premises were corrected in separate preserved attempts; no timeout or failure was accepted as parity.

Hosted checkpoint192 CI confirms the earlier enum test migration: the parent and all55 focused parents pass. Compiler failures fell from37 to36; there are no new failure or skip names. CI still fails overall. All124 CI artifacts were rehashed.

Other private work has advanced: the original112-source Gson application passes raw strict conversion; its project packaging, Go build and parity remain open. The implicit-lambda repair removes six compilation diagnostics from the original20-source Map reducer, leaving three inherited-method errors. Reader/runtime prerequisites pass native gates, while source integration controls expose further projection and wrapper failures. The array-null optimization has fresh reducer parity but is reserved for a separate milestone. None of these private changes is included here.

Next: retry the unchanged1220-source Netty workload with the matching normal compiler, continue Supplier/Gson TDD, and commit and push verified milestones. Full Netty, Gson, Supplier28, CI and round acceptance remain open.
