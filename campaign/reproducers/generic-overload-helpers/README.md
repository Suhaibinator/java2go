# Generic overload helper collision

`Overloaded.java` is a source-only Java 21 reproducer for two generic instance methods with the same name and distinct parameter types. It prints `ab` on the JVM; `expected.stdout` contains the exact output.

With the source snapshot that exposed the Gson failure, translation succeeded but generated `OverloadedFromJsonHelper` and `NewOverloadedFromJsonHelper` twice. `go build -mod=mod ./...` then failed on duplicate declarations in `j_p/Overloaded.go` at lines 49/74 and 53/78. The full Gson fixture remains the acceptance case.

To verify the Java oracle, compile `src/main/java/p/Overloaded.java` with Java 21 and run `p.Overloaded`.
