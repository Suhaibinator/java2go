# Reentrant synchronized-monitor parity fixture

Java intrinsic monitors are reentrant: a thread that owns an object's monitor
may enter another `synchronized` block guarded by that same object. Each
successful entry must later be matched by an exit, without blocking the owner.

This fixture evaluates both lock expressions exactly once and confirms that
they return the identical object. Java prints `OUTER=12`, enters the same
monitor again, completes both bodies, and prints `RESULT=123456`.

This formerly failed when generated Go associated the object with a
non-reentrant mutex and the nested acquisition deadlocked. The fixture now
passes and guards owner-aware reentrancy depth while retaining the observable
lock-expression order, balanced exits, object identity, and mutual-exclusion
semantics.
