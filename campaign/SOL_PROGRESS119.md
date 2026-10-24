# Milestone 119: Thread names and concurrent applications

Thread.getName now returns a stable canonical UTF16 reference. Canonical named constructors retain the supplied name reference and reject null; native name APIs retain their existing behavior. Producer-generated names use fresh String references rather than accidental interning. Two Thread oracle harnesses adapt their observations without modifying original Java or expected results.

The coordinator rebuilt matching race binaries, passed seven strict JDK21/generated-race probes and 28 native concurrency controls. Both unchanged round01 and round02 concurrent dependency applications pass all nine seed/repeat comparisons, all generated package and entry race builds, and20 additional stress observations each. Dependency implementations remain translated, with the fixtures’ frozen selected whole-class closures. Ordered outputs, exit status, stderr and declared files match.

A post-run verifier KeyError on the absent optional output-file map was retained and corrected; all original round01 observations were independently validated without changing observations or rerunning to mask behavior. Round02 includes its declared output file comparisons.

This does not establish whole concurrency API, setName, full Commons, full CI or whole-round acceptance. Full accepted campaign round remains18. Evidence and hashes are preserved under ignored .campaign/resume-20261002.
