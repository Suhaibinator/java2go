# Generated full-program example

These packages are generated from `testfiles/full_program`. The handwritten
`main.go` calls `app.Main()`. The module keeps Java package imports under
`com/acme` and uses the matching runtime from this checkout through the relative
`go.mod` replacement.

From the repository root, regenerate all source packages with:

```sh
go run ./cmd/java2go -w -sync -strict -init-go-mod -module com/acme -output generated/full_program testfiles/full_program
gofmt -w generated/full_program
```

Then, from this directory:

```sh
go mod tidy
go build ./...
go run .
```

The output must match
`testfiles/applications/existing_full_program/expected.stdout`. The
`TestApplicationParity/existing_full_program` test independently compares the
Java 21 program with freshly generated Go. Regenerate the whole tree when the
transpiler or runtime API changes; older generated APIs are not preserved.
