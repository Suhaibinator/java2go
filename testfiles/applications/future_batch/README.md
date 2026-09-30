# Future application regressions

These applications were written and run with OpenJDK 21 before implementation.
The initial transpilation of `future_batch` and `future_cancellation` failed to
build: `Submit` and `AwaitTermination` returned no value, value-returning lambdas
were lowered as Runnable, TimeUnit was undefined, and result-bearing submit was
unresolved. `thread_lifecycle` also failed on isAlive and timed join; the previous
runtime hung on join before start and silently allowed a repeated start.

`callable_forms` covers declared Callable lambdas/references, anonymous classes,
and named implementations. Adding its named implementation exposed a further
behavioral failure: submission returned null instead of 19. The type resolver now
follows implemented interfaces and superclasses to select Callable submission.

Each directory's expected.stdout is the Java oracle. The standard live-JDK
`TestApplicationParity` test compiles/runs Java, transpiles the entire application,
then builds/runs Go and compares stdout. `TestFuturesApplications` also compiles
and executes the generated code with Go's race detector. Runtime tests stress
shutdown/submission races, competing get/cancel/completion, cancellation of queued
and running tasks, positive/zero/negative/null/overflow timeouts, Java/native
failure causes, worker survival, queue growth, and thread lifecycle transitions.
