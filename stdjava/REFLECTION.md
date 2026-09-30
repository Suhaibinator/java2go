# Generated class reflection

The transpiler emits a registry when source code uses the supported reflection
operations. Java binary names identify generated classes, and class literals
share canonical identity with `Class.forName` and `getClass`.

Supported operations:

- `Class.forName(String)`, `getName()`, `getSuperclass()`, and
  `isAssignableFrom(Class)` for registered source classes.
- Public no-argument constructors of concrete, non-generic top-level or static
  member classes, through `getConstructor().newInstance()`.
- Public instance fields through `getField(name).get(object)` and
  `set(object, value)`, including inherited/hidden fields, boxing, primitive
  widening, nullable references, and final-field write rejection.
- Public zero-argument instance methods through `getMethod(name).invoke(object)`.
  Invocation selects overrides and carries the current Java execution token,
  including through synchronized methods. Primitive results are boxed, void
  results are null, and target exceptions are wrapped in
  `InvocationTargetException` with their cause preserved.
- `isAnnotationPresent(...)` for generated marker annotations explicitly
  declared with `@Retention(RetentionPolicy.RUNTIME)`, with `@Inherited`
  superclass traversal. Default CLASS and explicit SOURCE retention are omitted.
  The runtime `java.lang.Deprecated` marker is also supported.

Registration and class literals do not initialize Java classes. `forName` and
reflective construction use generated lazy initialization coordinators.
Initializer failures remain initialization errors; exceptions thrown inside the
constructor or invoked method are wrapped as invocation target exceptions.

This is a bounded reflection surface. Parameterized/varargs constructors and
methods, static members, generic/inner/local construction, accessibility
suppression, declared-member enumeration, annotation proxies/values, annotation elements/default values, and class loaders are outside this implementation.
Only generated packages linked into the Go program can register classes; loading
new bytecode or an otherwise unimported Go package by name is unsupported.
Unsupported member signatures are absent from the registry and produce
`NoSuchMethodException`/`NoSuchFieldException`; supplying arguments to an obtained
zero-argument member produces `IllegalArgumentException`. These descriptors do
not provide general JVM or Spring compatibility.

Explicit empty or null reflection API varargs arrays are accepted; a single
`(Object)null` argument retains its Java arity.

The four `runtime_metadata_*` applications under `testfiles/applications` provide
Java-oracle parity coverage. `reflection_test.go` covers runtime failure and
conversion boundaries.
