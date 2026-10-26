# Compiler regression repair checkpoint 234

This checkpoint repairs the seven compiler failures observed in CI 233. Protected overrides across packages now pass through the same visibility restriction in both planner paths. Generic-family discovery shares only current-batch AST node facts, retaining fresh ownership and semantic checks; callable-subclass rejection can stop before unrelated syntax. Lexical type-name inspection avoids allocating a split-result slice.

The historical allocator comparison is isolated from newly optimized discovery helpers. Its 19 helper bodies preserve commit `4f544674f1fdbb06d67240c4b4bf800c852dd49e` after test-local renaming. Actual production paths, generated AST/import assertions, frozen work limits and old-work sensitivity assertions remain unchanged.

Validation: 146 compiler parents and 184 subcases passed in both normal and race modes. Both original 4059 budget parents also passed with real atomic coverage and race instrumentation. golangci-lint 2.15 reported 0 issues. The original six nominal-hygiene cases run separately in each mode; the two original project-inventory cases run separately in race mode. Every case retains its parent setup and assertions, with unchanged 55-second native and 65-second supervisor caps; no aggregate pass is claimed for those partitions.

The first combined candidate's race allocation failure is retained: 6,688 bytes exceeded 6,656. The final one-line lexical allocation fix passes the same unchanged budget. The unsuccessful extra recursive-parameter change was removed. An earlier log-validator stop was separately corrected because an intentionally failing child was quoted in a passing negative-control test; no completed payload was repeated or credited as a repaired compiler failure.

This is a focused compiler milestone. Hosted CI for this commit, full Supplier 28 and Gson 112 behavior, Netty 1,220-source strict conversion and the round-boundary suite remain required. No dependency/application implementation replacements, generated-Go edits, source-name cases, assertion reductions or new skips are accepted. The campaign remains active with Sol agents only.

The private Netty query remains blocked: its normal-mode native calls increased from about 52.6 million to 114.0 million, and race mode still exceeds the original 55-second limit. Repeated positive callable queries need a complete discovery-owned structural inventory while standalone cold queries retain early rejection. This is an active follow-up, with no Netty query or application acceptance claimed.
