# Maven application fixture

`reactor` contains an aggregator POM and two production modules: `app` calls
`domain`, which depends on the separate `vendor:formatter:1.0` source project.
The application reads a copied production resource through `Files.readString`,
passes an argument to `main`, and calls an overloaded method named `main` in a
second class that also has an unselected application entrypoint. The test-only
source deliberately imports an unavailable JUnit type and must be excluded.

`TestMavenApplicationParity` in `e2e/maven_project_test.go` compiles production
sources with `javac`, runs the Java oracle, converts the reactor through the
public API, and builds/runs the generated Go module. Both print:

```text
customer:42
invoice-ready
```

Application-first baseline: the Java program compiled and produced that output;
the converter failed with `flag provided but not defined: -maven`. Implementing
project discovery then exposed Go's reserved `vendor` directory and the need to
marshal process arguments into the Java reference-array runtime representation.
The completed conversion passes the same Java/Go stdout and clean-stderr checks.

Resources are ordinary files under the generated module's `resources/` directory.
This fixture does not exercise `ClassLoader.getResource` or classpath resources.
