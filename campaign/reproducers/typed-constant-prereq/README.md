# Typed constant evaluator prerequisite

This additive Java 21 probe is separate from the frozen constant/init project
and all full campaign applications. It combines inherited and hidden constant
fields, final-local shadowing, narrowing and shift masking, constant String
folding, side-effecting null qualifiers, delayed class initialization, and a
source class named `String` distinct from `java.lang.String`.

The source class has both a true Java String constant and a final field of its
own `String` type. The latter is a runtime object, so its access must initialize
its owner. The inherited constants and their aliases must preserve their
declaration binding while owner initialization remains delayed.

Compile with the explicit JDK 21 javac, `--release 21 -encoding UTF-8`, then
run `prereq.typed.app.Main` in three fresh JVM processes for each input in
`inputs/seed-{17,41,97}.txt`. Exact expected stdout/stderr and exits are pending
an assigned oracle slot. No compiler, JVM, or transpiler command has run yet.
