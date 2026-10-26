# Concurrent batch service

A Java 21 Maven application with five source files across `model`, `service`, and
`app`. It uses the actual Commons Codec `Hex` implementation for every payload
and decodes each result to verify a UTF-8 round trip. A deliberately blank
task raises an exception; the encoded result travels through a typed future.

The first phase holds the only worker at a latch, cancels a queued future, then
releases the worker so a third task fails. The second phase starts three workers
behind one gate and processes six seeded jobs. All output follows submission
order. A synchronized ledger checks legal transitions, execution on a worker,
absence of an owner for the cancelled job, and exact completion counts. Latch
and future waits have bounded timeouts only as deadlock guards; no timing selects
an output order.

Seeds: `17`, `41`, `97`. Repeat each three times and compare exact stdout with
its Java oracle. Use JDK 21 at
`/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`.

The exact Java stdout for each seed is frozen in `expected.seed-*.stdout`;
`oracle.sha256` records the app, POM, manifest, eight upstream Codec source
files, binary oracle JAR, and snapshots. The campaign runner compiles the
selected upstream source closure and repeats every Java and Go run:

```sh
go run ./cmd/javacampaign -fixture testfiles/campaign/round01/concurrent
```
