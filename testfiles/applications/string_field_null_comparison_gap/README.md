# String-field null-comparison TDD fixture

This application proves that Java `String` fields are references at every point
in their lifetime. It compares an implicit default, an empty string, and an
assigned value with `null`.

Generated Go currently represents all three fields as `string`. The zero value
therefore loses the distinction between Java `null` and `""`; direct comparisons
with `nil` also fail to compile. The known-gap assertion pins that representation
failure until field storage preserves nullability. Explicit `null` field
initializers and assignments fail earlier for the same representation reason and
are covered by the implementation analysis for this fixture.
