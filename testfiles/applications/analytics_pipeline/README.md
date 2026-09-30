# Analytics pipeline parity fixture

This fixture is a deterministic, multi-package application rather than a
single-feature snippet. It parses an embedded event feed, validates malformed
rows, scores accepted events, aggregates segment metrics, performs a stable
generic ranking, and prints a byte-for-byte report captured in
`expected.stdout`.

## Coverage

- 16 Java source files across `app`, `common`, `model`, `parse`, `pipeline`, and
  `report` packages.
- Generic classes and interfaces (`ParseResult<T>`, `RecordParser<T>`), plus a
  bounded generic insertion sorter (`StableRanker<T extends Ranked>`).
- Abstract/concrete inheritance through `ScorePolicy` and
  `EngagementScorePolicy`.
- `List`, `ArrayList`, `Map`, and `HashMap` for event storage, rejection
  tracking, aggregation, and ranking.
- Arrays, enhanced and indexed loops, nested conditions, `while`, `switch`,
  ternaries, null checks, string normalization/splitting/comparison, integer
  parsing, and `Math.min`, `Math.max`, and `Math.abs`.
- Deterministic rejection ordering, ranking tie-breaks, hotspot selection, and
  an order-sensitive checksum. The fixture does not read stdin, files, clocks,
  randomness, locale settings, the network, or environment variables.

## Parity status

Passing. The parity harness compiles and runs the Java oracle, transpiles the
complete source tree, compiles and runs the generated Go module, and compares
their stdout byte-for-byte. This fixture remains a regression test for generic
interface instantiation, interface method dispatch, field-backed collections,
abstract-class companion interfaces, bounded generic calls, and qualified
cross-package local types.

## Oracle

`expected.stdout` was recorded from `javac`/`java` with UTF-8 source encoding.
It includes its final newline and is intended for exact comparison with both
the Java and generated Go processes.
