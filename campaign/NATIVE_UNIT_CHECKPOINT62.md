# Native compiler checkpoint62

This branch preserves the native compiler candidate whose fresh unit run removes all six failures introduced by the previous candidate: Function callback registration, List projection/nominal ownership, and four generated-API assertions. Assertion migrations retain the Java observations and strengthen receiver, execution context, projection, unboxing and ordering checks.

The implementation is the exact 1,983-file source snapshot `98f824f3059b14f9417cc4db5089f0d4125dda8b4a5a12e4a65a3c69719ce6b8`, private revision `4f00f0ef37b3eb18458715332e34d4fbd49af889`. Root verified every file after importing it into this managed checkout. This document is the only subsequent source change.

## Verification

The focused race gate passes 27 parents and 47 subtests, with no failures or skips. Independent audit: `071d87d04b318b369efd7b6885972da4e4c4d0e5a1d16b7f9c27e7222b67fad3`.

Fresh paired unit runs use unchanged selectors, JDK21, C locale, race checking and count=1:

| Snapshot | Passing tests/subtests | Failing | Skipped |
| --- | ---: | ---: | ---: |
| Baseline | 1,924 | 9 | 0 |
| Repaired candidate | 2,025 | 9 | 0 |

All baseline terminal identities and outcomes are retained. The 101 additional test/subtest identities all pass; no candidate-only failures remain. Eight retained binary build stamps and ten supervision/cleanup receipts were verified for each run. The nine shared failures concern Java ASCII stdout under the frozen locale; they remain failures, without an oracle or environment waiver.

Baseline receipt: `/private/tmp/java2go-native-ci-baseline-repaired-pair-terminal-sol-v26-irv3ienx/baseline-unit-terminal-receipt.json`, SHA256 `e7a7995851ff061672d39c7f88bec15daca25d2c7a441941477df053614e8b43`.

Candidate receipt: `/private/tmp/java2go-native-repaired-candidate-unit-terminal-sol-v2/receipt.json`, SHA256 `72676e3849bf2eb583bf517dfe6cfb4af4f55d588c53324e2d60450980d12364`.

Pair comparison: `/private/tmp/java2go-native-repaired-candidate-unit-terminal-sol-v2/paired-outcome-comparison.json`, SHA256 `51daf7bffd13a81e5103c6cce11a07a728cf0525c3608d8ce72060897bb615d3`.

Independent paired audit: `/private/tmp/java2go-native-six-paired-unit-sol-independent-comparison.json`, SHA256 `d9d79a4ca103c467230d5c64957f12a366cd83acc16525b4e2f2a13ebffd0c93`. It confirms all six previous candidate-only failures are repaired and all baseline outcomes remain unchanged. This supports a limited repair checkpoint while CI remains red.

## Remaining gates

This is a source checkpoint, not full round acceptance. The prior sixteen-group native application gate binds its earlier 1,980-file snapshot and must be rerun after implementation changes. The other CI jobs, coherent canonical runtime/compiler integration, full Gson and Codec applications, and Netty progression remain pending. A separately captured Function class-header shadowing challenge remains open. Parent race checking does not establish race checking for every nested generated application in older helpers.

Root owns managed Git publication. Sol workers retain isolated implementation and output directories; primary user changes remain preserved. Resume the active Sol-only campaign from the campaign ledger on `codex/adversarial-campaign`.
