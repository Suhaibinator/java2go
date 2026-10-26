# Scoped nested constructor counterexample

Preserved from the generic-helper naming regression. JDK21 prints `abc`; generated Go currently refers to undefined `ConstructInner` for `new Outer.Inner()` across the package cycle. This is a separate source-class construction blocker, not a substitute for full frozen Gson acceptance.

From the repository root:

```sh
probe=$(mktemp -d)
"$JAVA_HOME/bin/javac" --release 21 -d "$probe/classes" $(rg --files campaign/reproducers/scoped-nested-constructor -g '*.java')
"$JAVA_HOME/bin/java" -cp "$probe/classes" q.Bridge
go run ./cmd/java2go -strict -maven campaign/reproducers/scoped-nested-constructor -main-class q.Bridge -runtime "$PWD" -output "$probe/generated"
(cd "$probe/generated" && go build -mod=mod ./...)
```
