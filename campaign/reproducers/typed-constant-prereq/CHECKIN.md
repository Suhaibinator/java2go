# Additive typed-constant prerequisite check-in

Proposed repository location: `campaign/reproducers/typed-constant-prereq`.
This package preserves the original Java sources, README, POM, and seed inputs
byte-for-byte. It is a supplemental prerequisite, not a replacement for the
frozen original String constant challenge or any full campaign application.

JDK21 oracle status: validated. All nine fresh JVM runs (seeds 17, 41, 97,
three repetitions each) exited 0 with exact repeat-stable stdout and empty
stderr. The canonical streams and their hashes are in `oracle/`.

Transpiler status: UNRUN. No Go parity or candidate acceptance is claimed.
The original full challenges remain unchanged acceptance targets.
