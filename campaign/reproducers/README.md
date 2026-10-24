# Preserved compiler counterexamples

These independent compiler defects were discovered while repairing Map.putAll / Math.multiplyExact. The original sources remain as counterexamples; the status below distinguishes repaired cases from active failures. They do not replace the frozen application acceptance gates.

- `static-field-shadow`: repaired in the current working tree using storage-address helpers for shadowed same-package static fields. The JVM prints `7`; previously the source-qualified write changed the parameter instead. The unchanged full discovery case in `original-map-mutation/CampaignMapMutation.java` now also passes `TestCampaignStaticFieldShadowOriginalMutationJVMParity`.
- `signed-long-min`: repaired in the current working tree by folding integral unary literal chains with Java int/long widths. The original JVM value is `-9223372036854775808`; earlier Go output converted the positive token before negation and overflowed. Permanent `TestCampaignSignedUnaryLiterals` additionally checks underscored/suffixed literals, double-negation wraparound, bitwise complement, overload choice and boxing against JDK21. Focused tests pass; full integration verification remains pending.

Run from the repository root, with Java 21 selected through JAVA_HOME:

```sh
case_name=static-field-shadow # or signed-long-min
probe=$(mktemp -d)
"$JAVA_HOME/bin/javac" --release 21 -d "$probe/classes" "campaign/reproducers/$case_name/src/main/java/repro/Main.java"
"$JAVA_HOME/bin/java" -cp "$probe/classes" repro.Main
go run ./cmd/java2go -strict -maven "campaign/reproducers/$case_name" -main-class repro.Main -runtime "$PWD" -output "$probe/generated"
(cd "$probe/generated" && go run -mod=mod ./cmd/app)
```

Original discovery logs are `.campaign/map-multiply-red.log`, `.campaign/map-putall-mutation.log`, and `/tmp/campaign-map-mutation-convert.log` in the author workspace; these logs are diagnostic artifacts, not required inputs to reproduce the defects.

`enum-static-concat/CampaignStaticEnum.java` preserves an independent enum static-field string-concatenation inference failure discovered by the storage regression. JDK21 `CampaignStaticEnum.run()` returns `9:5`; generated Go incorrectly invokes `StringJava2goExecution` on the `int32` field. The exact source remains here; `TestCampaignStaticFieldShadowEnumJVMParity` isolates the enum storage contract with a primitive result. To reproduce, append a Java main wrapper calling `CampaignStaticEnum.run()` for the JVM oracle, and transpile the original file with `go run ./cmd/java2go campaign/reproducers/enum-static-concat/CampaignStaticEnum.java`. This remains a separate compiler blocker.
