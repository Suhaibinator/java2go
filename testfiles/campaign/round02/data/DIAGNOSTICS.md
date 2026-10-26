# First strict differential result

The frozen round-two Gson challenge passes the offline Maven build, selected
whole-class source-only JDK 21 compilation, and all nine binary/source JVM
oracle comparisons. The first translator failure is in the real Gson 2.14.0
`com/google/gson/internal/ConstructorConstructor.java`, line 235. A
`line_comment` appears between operands of a parenthesized boolean
expression, and expression conversion treats it as an operator.

The report is
`.campaign/runs/20260928T051546Z-985480644/report.json`. A supplemental
JDK-compilable reproducer at `/tmp/CommentExprRepro.java` has SHA-256
`5f8a6cab64195f5bd0b9d3aa931e0719368ae814b4aa4ac942a1e1b045d273c9`.
Running `go run ./cmd/java2go /tmp/CommentExprRepro.java` reaches
`StrToToken(line_comment)` at `transpiler/expression.go:1460`. The full Gson
workflow remains the acceptance target.

After the comment-expression repair, the next full run reached
`com/google/gson/internal/GsonTypes.java:153` and reported unsupported
`assert_statement` for `assert bounds.length == 1;`. The run artifact is
`.campaign/runs/20260928T052208Z-1124070720/report.json`; concurrent
implementation edits invalidated its overall fingerprint, but the strict
transpile-stage diagnostic is recorded. A supplemental JDK-valid reproducer
at `/tmp/GsonAssertRepro.java` has SHA-256
`7b63da1015dab287fc6c9fe0c0a223240b755c5d4c67736a172b71e276b50741`.

After assertion support landed, the full library conversion reached generated
Go parsing and failed in `GsonBuilder.go:392`: the first line of a multiline
`@InlineMe` annotation became a Go comment, while its `replacement = ...`
and `imports = ...` continuation lines remained raw Go declarations. The
upstream source is `com/google/gson/GsonBuilder.java:577-580`; the full
report is `.campaign/runs/20260928T053108Z-3915107237/report.json`.
`/tmp/MultilineAnnotationRepro.java` is a supplemental JDK-valid reproducer
with SHA-256
`0194f1cb0e919aa3304daab01d8686eab55d24aecdfff50c18dd308a6a5cfb33`.
