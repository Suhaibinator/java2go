# Reentrant synchronized-monitor known gap

Java intrinsic monitors are reentrant: a thread that owns an object's monitor
may enter another `synchronized` block guarded by that same object. Each
successful entry must later be matched by an exit, without blocking the owner.

This fixture evaluates both lock expressions exactly once and confirms that
they return the identical object. Java prints `OUTER=12`, enters the same
monitor again, completes both bodies, and prints `RESULT=123456`.

Generated Go currently associates the object with a non-reentrant `sync.Mutex`.
The nested acquisition blocks the only goroutine, so Go's runtime reports a
deadlock immediately; the test does not rely on the parity harness's 60-second
timeout. The fixture pins the fatal diagnostic until the monitor runtime tracks
owner/reentrancy depth while retaining mutual exclusion between threads.
