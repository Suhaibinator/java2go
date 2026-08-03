# String-field null-comparison TDD fixture

This application proves that Java `String` fields are references at every point
in their lifetime. It compares an implicit default, an empty string, and an
assigned value with `null`.

This formerly failed when generated field storage lost the distinction between
Java `null` and `""`, and direct comparisons with `nil` did not compile. The
fixture now passes and guards all three states—implicit default null, empty
string, and assigned nonempty value—so field nullability and comparison lowering
cannot silently regress.
