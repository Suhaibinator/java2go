# Canonical String candidate checkpoint19

This branch preserves the canonical String implementation and its unchanged regression fixtures. It is a work-in-progress source checkpoint, separate from the accepted campaign implementation. Full accumulated CI and dependency-application acceptance remain pending; the full Gson application remains required and failing. Newer native generic/import and erased-collection repairs are separate candidates and are not included here.

The source matches the frozen 1,880-file candidate manifest `faa2c898478a7052d487a21a0a9860796049efbd781d61b2833854aa3715b52d`, except the two campaign tracking documents and this publication note. No implementation or fixture bytes changed for publication.

Verified behavior includes immutable UTF16 String references, the exercised null/identity/isolated-surrogate contracts, canonical Integer.parseInt, source text dispatch, and Throwable message constructors/getters/virtual callbacks. Generic String-bound constructor selection and mixed native/canonical initCause rejection preserve the exercised message, cause, execution and ordering contracts.

Evidence on this exact implementation:

- All five unchanged JVM-first Throwable compiler controls pass strict transpilation, all-package and entry race builds, and exact output comparison.
- The unchanged larger String workflow passes all 16 JVM/normal-Go observations and all 10 race observations, including evaluation order, bounds and isolated UTF16.
- Four supplemental all-package builds pass: both generated projects in normal and race modes. Generated source and source implementation hashes remain unchanged.
- Runtime component gates include the mixed callback rejection oracle and 38 existing native Throwable compatibility tests.

The coordinator ran the full workflow from clean outputs. An independent agent audited raw observations, generated sources, binary metadata, toolchain hashes, and cleanup receipts. The full workflow report SHA256 is `d983cd248e9596fad684ad1e080060e65050dbe38d654c1cf949424ab4620b05`; independent semantic audit is `71ff84922c164d07b8a82b8753b357028f3ee882fad79302d0f502a9aae5d8ed`; supplemental package audit is `8ea3ec39dfabfd284df1ea6a709d4cf41c073d6750e17207bd488dea1bb25179`. Verbose reports are retained outside tracked source in the campaign checkpoint archive.

The five compiler controls can be selected with:

```sh
go test -race -count=1 -timeout=300s ./transpiler -run '^TestCampaignThrowable(ReferenceCompiler(Storage|Virtual|NullCause)|Constructor(BoundString|BareNullPair))JDK21$'
```

Use the installed JDK21 explicitly for both Java compilation and execution. These focused results establish the stated corpus behavior; they do not establish full canonical runtime coverage, full Gson or Netty support, performance superiority, or completion of the campaign.

## Canonical integration and lint checkpoint

This component composes native iteration/collection and canonical String prerequisites, source Object text dispatch, boxed-wrapper adapters, List text behavior and boolean inference for instanceof. The coherent1,934-file source passed all16 focused stages with independent raw audit `5cd1a769ce0a448e9908233e420615f5f37078b060a30aadcb72e763554f7ffd`.

A fresh original22-stage gate then passed five version/build stages and failed lint with eight issues; the remaining16 stages were unrun. The subsequent lint-only derivative removes seven unused helpers and replaces two syntactically identical calls in the empty-array test with separate variables, retaining the original identity observation and adding array descriptor/length checks. No surviving production function body, Java fixture or oracle changed. The resulting1,933-file source passes the original lint command, original charsJDK1 and canonical String5 tests. Independent terminal audit: `4a01e3cd79d5e25010537569e8f1727e1c4d6234c6c803bcec14a2fef3685dc0`.

A fresh full22 run on the lint derivative passed eight stages, then exposed two obsolete astutil String shape assertions expecting raw Go strings. Thirteen later stages were unrun. The test adaptation and independent audit remain pending; no full acceptance is claimed.

Final tested source manifest: `32e5dcbfb68df7469ff7cc09d0205ac156b00566f2c20b4f0b9ca2f102f87f31`. Root publication verifies all1,931 nonledger candidate files plus retention of the three previously published metadata/probe files. Two current tracking files are preserved and updated. These are component results; the new originalfull22 run and full Gson remain required. Number/Enum, DateFormat and newer collection repairs are separate candidates awaiting coherent integration. Checkpoint18 remains the latest full compatibility acceptance.
