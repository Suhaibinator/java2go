# String-reference prerequisite probe

Source preparation only. This is a small, JDK-only reproducer for the valid
ConcurrentMap schedule's first generated-Go divergence. It does not modify
that full schedule, a frozen campaign fixture, or the implementation. Do not
run javac, Java, Maven, Go build, or the transpiler until a shared execution
slot is assigned; no expected-output snapshot exists yet.

The probe observes literal interning across a static field and method, separate
`new String` allocations with equal text, aliases through `String` and `Object`,
`intern()` returning the literal's reference, a char-array constructor, and
null comparisons. `HashMap` must use String value equality for keys despite
the distinct references; it also checks a null key. A `fresh` helper increments
state and stores its last allocation, so a final identity comparison checks
that the right operand is constructed exactly once and remains distinct.
These observations are printed in declaration and operation order, with
assertions before output; there is no unordered iteration or timing dependency.

Pending JDK 21 validation after slot release: compile with `javac --release 21`,
run fresh JVMs, compare exact stdout/stderr/exit, and record source hashes
before and after. Then run strict source transpilation and compare generated Go.
Do not shorten the source to make compilation pass.
