# String constant and initialization prerequisite

This small four-package Java 21 source probe targets compile-time String and
primitive constant expressions, field and local aliases, integer overflow and
shift folding, runtime concatenation identity, static constant access through
side-effecting null qualifiers, class initialization order, and a source class
named `String` distinct from `java.lang.String`.

It is separate from the frozen String ABI probe and campaign applications.
Use the explicit JDK 21 javac with `--release 21`, then run
`prereq.constant.app.Main` in a fresh JVM three times for each seed in
`inputs/seed-{17,41,97}.txt`. Compare exact stdout, stderr, and exit code.
Compilation and oracle execution are pending an assigned execution slot;
expected output must come from that JDK oracle.
