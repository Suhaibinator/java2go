# Lazy class-initialization known gap

Java class initialization is demand driven. Starting this application's main
class initializes `LazyClassInitializationApplication`, whose initializer
records `M`; it does not initialize the otherwise unused `DormantClass`, even
though both classes share a source file and package. The Java oracle therefore
prints `TRACE=M` and `MARKER=1`.

Generated Go currently lowers class initializers into package-level `init`
work. That eagerly initializes `DormantClass` too, making its `D` side effect
observable before `main`: it prints `TRACE=MD` and `MARKER=1`. This fixture
pins that exact output divergence until translated classes have Java-compatible
first-active-use initialization.
