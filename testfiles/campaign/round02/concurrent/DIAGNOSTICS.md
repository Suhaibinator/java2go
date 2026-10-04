# First strict Go discrepancy

The frozen Java oracle and the explicit eight-file Commons Codec source-only
oracle produced byte-identical stdout and `results.txt` for seeds 17, 41, 97,
each repeated three times. JDK 21 is
`/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`.

Initial strict command:

```sh
go run ./cmd/javacampaign -fixture testfiles/campaign/round02/concurrent
```

First report: `.campaign/runs/20260928T051529Z-444414224/report.json`.
The failure stage is `go-build-all`, after offline Maven, selected-source javac,
binary-jar Java, source-only Java, and strict transpilation succeeded. The
first generated package errors include unresolved `ByteArrayInputStream` base
and constructor for `TrackedInput`, `ThreadLocal`, inherited `input.read`,
`ByteArrayOutputStream.write`, `Arrays.equals`, and `Files.readAllBytes`, plus
`String.indexOf('|')` passing a char to a string-only helper. The fixture was
not changed after recording this discrepancy.

Temporary focused reproductions, outside the frozen fixture:

- `/tmp/java2go-threadlocal-probe`: a single worker observes `ThreadLocal`
  `get`/`set`/`remove` and reinitialization; JVM output `7:0`. Its generated
  build fails on unresolved `ThreadLocal`.
- `/tmp/java2go-string-array-probe`: UTF-8 `indexOf(char)` and byte-array
  equality; JVM output `5:true:false`. Its generated build fails on the char
  argument and unresolved `Arrays`.

## Acceptance after implementation repairs

The fixture inputs and snapshots above remained unchanged. The full command
below passed with race detection and 20 extra isolated Go executions:

```sh
go run ./cmd/javacampaign -fixture testfiles/campaign/round02/concurrent -race -stress-runs 20
```

Passing report: `.campaign/runs/20260928T052554Z-1134746446/report.json`.
It contains nine exact Java/Go seed-repeat observations and 20 exact stress
observations. Offline Maven, selected-source javac, Java oracles, strict
transpilation, race-enabled Go package build, and output-file comparisons all
passed. `shasum -a 256 -c testfiles/campaign/round02/concurrent/inputs.sha256`
reported no changed input or oracle file.
