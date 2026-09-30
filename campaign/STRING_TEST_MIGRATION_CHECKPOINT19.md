# Canonical String test migration publication

This scoped milestone updates seven original test files to observe canonical `*JavaString` values and their generated AST shapes. The original Java programs, JVM expectations, assertion coverage and production implementation are preserved. String observations reject null and compare every UTF-16 unit. Boxing diagnostics retain Go vet checks.

The publication source is an exact archive of `28e50af626385b9c2b071fe6d07ea3cad19a751f` plus the seven independently accepted test files. Its 1,936-file manifest is `d2a51cac7c14407b0ab33d438b36bfd7d18f0d272af06398924ab4e3dd2b617c`; three previously published probe/document files and all tracking records are retained.

Fresh composition verification passed all eight astutil parents (51 total test keys) and all 42 selected AST/boxing parents (53 total test keys), with `-race`, `-count=1`, no failures or skips, stable source and recorded regular tool files, and supervised stage cleanup. Independent audit `c8cc22951717c079d4d843a43e5dccd101a7d58a2dc5b109d8ef0aa1c52de394` verifies exact Git ancestry, source/file correspondence, raw test selections and results, and Java input retention. Historical red and vet failures remain recorded in the campaign ledger.

The outer runner itself has no aggregate supervision receipt; both test stages have bounded supervision. Recorded tool manifests cover regular files, excluding non-executable legal/license symlinks. Existing generated test helpers use plain Go tests; `-race` on parent tests does not claim child race instrumentation.

This milestone does not accept full Gson or cumulative CI. Their original gates remain required.
