# Routing engine parity fixture

This fixture models a small route-planning service rather than an isolated
language feature. It builds a directed network, runs repeated edge relaxation
with deterministic path tie-breaking, applies interchangeable cost policies,
handles an unreachable destination, and prints a compact report plus an
order-sensitive checksum.

## Coverage

- 9 Java source files across `app`, `algorithm`, `model`, and `policy` packages.
- Cross-package constructors, static factories, encapsulated fields, and method
  calls.
- Interface-based policy dispatch with two concrete implementations.
- Reference and primitive arrays, indexed loops, nested conditions, early loop
  termination, ternaries, and compound boolean expressions.
- String concatenation, equality and lexical comparison intrinsics.
- Multiple algorithm scenarios whose paths, costs, hop counts, unreachable
  handling, and aggregate checksum all affect the oracle.

The application has no stdin, files, clocks, randomness, locale-sensitive
formatting, network access, or environment-dependent behavior.

## Oracle

`expected.stdout` is the exact UTF-8 output produced by the JDK, including its
final newline. The differential harness first verifies that Java still matches
this snapshot and then requires the generated Go application to match the same
bytes. This fixture currently has full parity and is marked `passing` in
`fixture.json`.
