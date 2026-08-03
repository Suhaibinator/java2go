# Lazy class-initialization parity fixture

Java class initialization is demand driven. Starting this application's main
class initializes `LazyClassInitializationApplication`, whose initializer
records `M`; it does not initialize the otherwise unused `DormantClass`, even
though both classes share a source file and package. The Java oracle therefore
prints `TRACE=M` and `MARKER=1`.

This formerly failed when generated Go eagerly initialized `DormantClass`,
making its `D` side effect observable before `main` and producing `TRACE=MD`.
The fixture now passes and guards Java-compatible first-active-use
initialization: the dormant class must remain uninitialized and the exact oracle
must stay `TRACE=M` and `MARKER=1`.
