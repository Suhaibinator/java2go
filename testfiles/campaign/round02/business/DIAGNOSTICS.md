# First frozen campaign run

Report: `.campaign/runs/20260928T051409Z-56114146/report.json`. The implementation SHA-256 was `ce8625bac44af3cd2a72bf52e0f3603977fa96d29aec171c16db1e216bfd7d09` both before and after. Original-POM Maven compile, exact dependency-source JDK compile, all nine JVM oracle observations, and strict transpilation passed.

The earliest failure was `go-build-all`. Generated `Coordinator.go` called `depot.balanced` although the public Java method was emitted as `Balanced`. Generated `Document.go` returned a `*Shipment` from the static `Document<String>` factory without converting to the generic base reference. The frozen application and dependency implementations remain intact.
