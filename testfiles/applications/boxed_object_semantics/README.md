# Boxed object semantics

This deterministic application checks Java 21 wrapper behavior across three
packages. `expected.stdout` is the exact output of the Java 21 program. The fixture
is required to pass transpilation, Go compilation, and output parity without a
known-gap exemption.

The application exercises all eight nullable wrapper fields, arguments, generic
returns, array defaults, and Object views; Number bounds and mixed numeric generic
inference; guaranteed cache identity and fresh wrapper constructors; immutable
aliases after increment and compound assignment; equality, hashing, ordering,
signed zero, NaN, and numeric accessor boundaries; primitive versus wrapper class
literals and `TYPE`; method and constructor overload
phases; null-unboxing and receiver evaluation order; and fixed/expanded varargs
array identity, mutation, null arrays, and covariant store checks.

The library package checks boxed map/set/concurrent-map keys and retained identity,
heterogeneous collection queries, `List.remove` overloads, primitive/reference
stream conversions, distinct, boxed and downstream collectors, Boolean partition
keys, and Optional null behavior. Output never depends on collection iteration
order: the only iterated key set contains one key.
