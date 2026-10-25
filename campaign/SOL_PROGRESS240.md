# CI regression repair checkpoint 240

Public239 introduced concrete List iteration and record-cast regressions in three retained dependency applications, plus repeated AbstractList header attestation that exceeded the existing import-work budget and contributed to a cumulative compiler-shard timeout.

The repair preserves typed native List iteration when no source AbstractList projection can flow, and supplies nominal identity for records when erased collection reads require it. Records retain their java.lang.Record parent and hygienic accessor names. A conservative superclass spelling filter rejects unrelated candidates before expensive attestation; possible candidates still undergo the original ownership, nominal and constructor proofs.

Verification on the frozen repair source:

- Three unchanged dependency applications: nine stable JVM/Go comparisons and twenty race stress runs each, including declared output files.
- Eight new compiler controls and retained record, collection, import and AbstractList controls: 85 real parents per normal/race mode, covering all 84 parents selected by actual Go AST discovery.
- Full runtime: 947 parents and 3,030 subtests per normal/race mode. All 2,679 compiler baseline parents retained in the 2,687-parent registry.
- All 17 original process/work/coverage parents pass with atomic and race-atomic coverage at their original budgets. Lint reports zero issues.

Six empty-selector qualification mistakes are retained in raw evidence and excluded from credited test results. The 196 nonempty qualification stages and separate binary metadata execution are recorded outside tracked source. Source and actual evidence receive independent review before publication.

This is a repair of the exercised regressions. Public239's incomplete hosted compiler observations are not relabeled as passing. Hosted CI on this new head remains required. Full Gson112, generic cross-instantiation record erasure, universal record reflection and Netty application acceptance remain open. No application fixtures, existing assertions, generated Go, strict-mode behavior or time limits were weakened. The anonymous-capture, Supplier, reflection and Netty candidates remain separately qualified private work.
