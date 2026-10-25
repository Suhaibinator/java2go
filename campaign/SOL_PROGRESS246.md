# Record constructors and reflective parameter counts

Written, compact and delegating record constructors now use the common constructor pipeline. Canonical constructors are matched by signature, compact bodies run before component assignment, and delegated calls preserve argument order and execution context. Record initialization uses the existing lazy class coordinator, including passive constructor references.

`Constructor.getParameterCount()` reports formal parameters, including one formal parameter for a varargs declaration. Reflective record construction remains tied to the exact registered source declaration.

Verification: 39 successful stages cover normal and race modes, lint, and independent actual review. Each mode passes 86 compiler parents, 20 selected runtime reflection parents and all 31 symbol parents. Ten record programs per mode exercise whole JVM/Go comparisons, constructor ordering and exceptions, source-ownership refusals, initialization and reflection interactions. All 2,749 existing compiler test names remain in the 2,763-name compiled inventory; listing is not a full execution result. Earlier failed runs remain recorded.

This is a prerequisite milestone. Full Gson and Netty applications remain unaccepted. Generic record reflective callbacks, component annotation propagation and reflective record static state remain explicit boundaries.

The campaign continues with frozen dependency applications and the existing original execution limits. Shared build cache cleanup happens at quiescent batch boundaries, preserving dependency downloads, source, logs and retained comparisons.
