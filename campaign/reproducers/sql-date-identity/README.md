# SQL Date nominal identity: retained JVM mismatch

Status: **RED**, captured 2026-09-28. This is a focused prerequisite reproducer,
not a frozen challenge acceptance or a claim of SQL date support. Production was
not changed while capturing it.

JDK21 compiles the exact checked-in source with `javac --release 21`. Its stdout,
without a final newline, is preserved in `expected.stdout`:

```
java.sql.Date:java.sql.Date:true:java.sql.Date
```

Strict Maven translation and the generated Go executable both exit successfully,
but stdout in `observed-go.stdout` is:

```
java.util.Date:java.util.Date:true:java.util.Date
```

The imported `Date.class` incorrectly receives the util.Date descriptor. Both
`new Date(0L)` with `import java.sql.Date` and the fully qualified
`new java.sql.Date(0L)` incorrectly emit `stdjava.NewDate(int64(0))`. The util.Date
instanceof succeeds in both Java and Go, so it does not expose the lost subclass
identity by itself.

Root causes are separate: `dateTimeRuntimeTypeID` ignores explicit imports for a
simple name, while `tryConstructorIntrinsic` indexes constructor registrations by
stripped simple name and passes no original owner to the plain generator. Merely
rejecting qualified SQL names in the type helper therefore cannot fix constructor
selection. The next repair must resolve canonical owners consistently before type,
class-literal, constructor, and method mapping. SQL Date must remain a distinct
Java nominal class with its real util.Date superclass relationship; unsupported
SQL services must not silently acquire util.Date identity.

Reproduce from repository root with JDK21 and a fresh temporary output directory:

```sh
"$JAVA_HOME/bin/javac" --release 21 -d /private/tmp/sql-date-classes campaign/reproducers/sql-date-identity/src/main/java/repro/Main.java
"$JAVA_HOME/bin/java" -cp /private/tmp/sql-date-classes repro.Main
go run ./cmd/java2go -strict -maven campaign/reproducers/sql-date-identity -main-class repro.Main -runtime . -module example.test/sqldate -output /private/tmp/sql-date-generated
(cd /private/tmp/sql-date-generated && go run -mod=mod ./cmd/app)
```

This is directly relevant to unchanged Gson SQL adapters: SqlDateTypeAdapter and
SqlTypesSupport construct fully qualified `java.sql.Date` and compare its Class
identity. DateFormat/SimpleDateFormat and Timestamp/Time remain separate missing
services, not substitutes for this nominal correctness prerequisite.

## Owner-resolution repair boundary

The next repair preserves the original JVM oracle and the historical wrong Go
observation above. Strict translation now rejects the SQL constructors with an
unsupported canonical `java.sql.Date` owner diagnostic, rather than substituting
util.Date. Imported, qualified, and wildcard SQL class literals independently
retain the correct name. This is a fail-closed compiler improvement; the full
program remains pending genuine SQL Date runtime support.

## Genuine SQL service acceptance

The same unmodified source now passes strict Maven translation and generated Go
execution with stdout exactly equal to `expected.stdout`, empty stderr, and exit
status 0. The historical failing observation above remains preserved. This
accepts this focused reproducer, not the complete Gson application.

The runtime now has distinct SQL Date, Time, and Timestamp classes sharing the
util.Date reference protocol. Focused JDK21 differential tests cover nominal
identity, base references and arrays, millisecond/nanosecond precision, asymmetric
equality, null and boxed arguments, local-zone text conversion, parsing failure
order, and Calendar.setTime polymorphism. DateFormat/SimpleDateFormat, java.time
conversions, deprecated field APIs, and source-defined Date subclasses remain
separate prerequisites.
