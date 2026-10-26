# Round 02 concurrent resource workflow

The job plan is a UTF-8 resource copied into each isolated run. A serial executor
holds its worker at a latch while one queued future is cancelled; a later task
fails with a blank body, and recovery succeeds on the same worker. A second
executor starts three workers behind a gate and processes five plan entries.
Results are collected in submission order and written to `results.txt`.

Each task copies bytes through a tracked input stream, uses the actual Commons
Codec 1.22.1 `Hex` implementation to encode and decode them, and removes its
`ThreadLocal` context in `finally`. The synchronized ledger asserts legal
states, distinct contexts on reused and concurrent workers, worker ownership,
and exact stream open/close counts. No sleep or thread-name ordering determines
output.

JDK 21 binary-jar and eight-file source-only oracles matched exactly for seeds
17, 41, and 97, each repeated three times. `oracle.json` freezes stdout,
stderr, exit code, and results-file hash; `inputs.sha256` records the app,
resource, POM, manifest, dependency sources, jar, and snapshots.

Run from repository root:

```sh
go run ./cmd/javacampaign -fixture testfiles/campaign/round02/concurrent
```
