# Runtime metadata wiring

This application loads a generated service using its Java binary name, obtains
its public no-argument constructor, writes and reads a public field, invokes a
public zero-argument method, and checks ancestry and the runtime `@Deprecated`
marker. `expected.stdout` was captured with OpenJDK 21 before implementation.

Application-first evidence:

- The initial Java application produced the committed oracle.
- The original generated Go failed compilation with `undefined: Class` and a
  missing `isAssignableFrom` method on `*stdjava.Class`.
- The implemented fixture is registered with the application parity harness,
  which compiles/runs Java and generated Go and compares their output and exit.
