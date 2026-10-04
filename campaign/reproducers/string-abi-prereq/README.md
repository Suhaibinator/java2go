# String ABI prerequisite

Small Java 21 source oracle for String identity, class initialization, valueOf,
builder results, UTF-16 code units, defensive copying, and String switch
evaluation. It is separate from the frozen campaign applications and has no
external dependencies. No expected output is recorded until the JVM oracle is
run in the assigned execution slot.

Compile with `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/javac`
using `--release 21`, then run `prereq.string.app.Main` with inputs in
`inputs/seed-{17,41,97}.txt`. Use fresh JVM processes for three repetitions of
each seed and compare exact stdout, stderr, and exit code. Compilation and
oracle execution are deliberately deferred until an execution slot is assigned.
