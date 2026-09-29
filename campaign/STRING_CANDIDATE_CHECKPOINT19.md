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
