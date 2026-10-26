# Matched Java/generated-Go list retention

**Measured result:** after removing all entries from a list kept alive across repeated batches, generated Go retains the last batch's array payloads. Java `ArrayList` does not materially retain them. Explicitly clearing the already-empty list releases the Go payloads. This is post-GC live-heap evidence, not a throughput/latency result, a measured optimization, or a campaign gate.

The [shared Java source](source/ListRetention.java) is compiled by JDK21 and transpiled unchanged to Go. Each cycle creates 64 `byte[]` arrays, writes/checks both end markers, adds them to a list, and removes them from the tail while accumulating a checksum. The static list stays live across four fill/remove cycles. Four small preliminary cycles warm this code path, then reset establishes the measured baseline. Runtime-specific harnesses force two collections before each live-heap snapshot. They print identical semantic output separately from JSON memory instrumentation. No generated application logic is patched by the harness.

Three fresh Java/Go process pairs ran at each batch size, alternating process order. Every pair matched all stdout bytes and exited successfully:

```text
round=0 checksum=5376 size=0
round=1 checksum=7680 size=0
round=2 checksum=10112 size=0
round=3 checksum=8704 size=0
cleared size=0
```

| Batch payload | Java median post-remove delta | Go median post-remove delta | Java removed-minus-cleared | Go removed-minus-cleared |
|---|---:|---:|---:|---:|
| 64 × 128 KiB = 8 MiB | 90,184 B | 8,392,224 B | 56 B | 8,392,208 B |
| 64 × 256 KiB = 16 MiB | 90,128 B | 16,780,832 B | 0 B | 16,780,816 B |

Post-remove deltas use each process's baseline. Removed-minus-cleared compares its final removal snapshot to its later clear snapshot, avoiding baseline formatting/class-initialization overhead. The Java ~90 KiB delta survives clear and does not scale with payload; do not attribute it to retained array payloads. The Go delta after clear is 0–128 B across the six processes. Go retains payload bytes plus roughly 3.6 KiB of array wrappers/list storage; Java and Go object layouts are different, so these are heap-lifetime measurements, not exact object-size comparisons.

Go's retained payload remains roughly constant across four cycles of each process, rather than growing every round. Refilling this same list overwrites prior stale slots. The problem is one maximum recent batch retained by each live emptied list; growth depends on how many such lists remain reachable. This is especially relevant when a service keeps empty reusable lists in long-lived objects or pools. It does not show an unbounded leak for a single reused list.

All twelve measured processes completed in under 0.2 seconds; each had a 30-second hard timeout. Those elapsed times are safety records only. The workload does not warm JVM JIT to steady throughput; no startup or speed conclusion is valid. Heap results come after explicit collection of otherwise unreachable arrays and repeat across sizes/processes.

## Source-level cause and general fix

At measurement time `stdjava/List.RemoveAt` returns the removed value and shortens the backing slice using append without clearing the vacated last slot. Go's collector still traces references in the allocated backing array beyond the visible slice length. The deterministic Go probe in the parent directory independently finds 32 non-nil stale slots in an emptied 32-element list.

The installed JDK21 source at `lib/src.zip!/java.base/java/util/ArrayList.java` confirms `fastRemove` shifts entries and assigns null to its vacated slot. This agrees with the measured absence of retained payload. No web or library-version assumption is needed.

Proposed general replacement body, **not applied**:

```go
old := l.elements[index]
last := len(l.elements) - 1
copy(l.elements[index:], l.elements[index+1:])
var zero T
l.elements[last] = zero
l.elements = l.elements[:last]
return old
```

The initial indexed read preserves current bounds-error timing. Copy preserves survivor order; the returned reference retains its identity; zeroing occurs only in the now-unused slot. Use the Go zero value even for nullable Java string representations: this is unreachable list storage, not a visible Java element. Keep capacity for reuse. The change remains O(n) for middle removals and O(1) for tail removals, and needs no added allocation.

Acceptance criteria for a runtime-owner implementation:

- Existing Java list removal/order/overload/exception parity remains green. Cover head, middle and tail removal, empty and invalid indices, returned identity, duplicate values, and null entries.
- A deterministic backing-storage check confirms the vacated slot is zero after every removal; include pointers and a value struct containing a reference. Do not rely on nondeterministic finalizers as the only proof.
- Rebuild and rerun this shared-source probe unchanged. All semantic stdout bytes must still match Java. At both batch sizes, post-remove minus post-clear should lose the full payload component; allow less than 1% of payload as a conservative instrumentation allowance, not an exact zero requirement. Repeated cycles should plateau near list-capacity/runtime bookkeeping.
- Keep the emptied list reachable throughout GC. Do not replace removal with `Clear`, discard the list, or keep the last returned array in the harness: those would test a different lifetime.
- The deterministic existing probe's `stale_list_references` should become zero. No throughput improvement is required to accept the retention fix. Any later throughput claim requires separate isolated, warmed trials against both prior Go and Java.

## Reproduction and raw evidence

Coordinate a clear performance window before running:

```sh
python3 performance/runtime/list_retention/run.py
```

Optional dimensions: `--trials 3 --rounds 4 --count 64 --sizes 131072 262144`.

The runner builds the compiler, strictly transpiles the shared source, compiles both variants, compares exact semantic output, and saves raw commands/stdout/stderr/generated files/hashes under `.campaign/performance/list-retention-<timestamp>/`. It does not invoke or change the campaign gate. Measurement instrumentation is intentionally separate from the Java application source because Go and Java expose different GC/heap APIs.

This run's evidence: `.campaign/performance/list-retention-20260927T210737/results.json`, with individual process files and `commands.json` beside it. Relevant fingerprints:

- Shared Java source SHA256: `71f10421f36639ec6fc4d59541222a4dfd0035a426deba734f88c0bd498dcbe3`
- Generated application SHA256: `5f084f5aa4433cc6a73436d38350e569e5820526bace79e3fa75a81d443f06a5`
- Inspected `stdjava/list.go` SHA256: `c0422d1506729c4d65ef92882b1f71c06f22bb4509d1e0b5d0e65e8106e3063a`

Toolchains: Go 1.27.1 Darwin arm64; Oracle JDK21.0.6+8 at `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`. Java uses SerialGC for synchronous diagnostic collection, `-XX:ActiveProcessorCount=2 -Xms32m -Xmx256m`. Go uses `GOMAXPROCS=2 GOMEMLIMIT=256MiB`, default GOGC=100 (the runner now sets 100 explicitly). Both have the same requested CPU count and nominal 256 MiB budget, but Java Xmx and Go GOMEMLIMIT are different controls; no total RSS/capacity comparison is claimed. JVM used heap is `totalMemory-freeMemory`; Go used heap is `MemStats.HeapAlloc`; paired within-runtime removal/clear differences support the lifetime conclusion despite accounting differences.
