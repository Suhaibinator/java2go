# Annotation-driven component wiring

A runtime `@Component` marker selects a service from binary-name candidates.
Only the selected service is constructed, configured through a field and invoked.
The application also verifies default CLASS and explicit SOURCE retention are
excluded, `@Inherited` follows superclass ancestry, and a non-inherited marker
does not appear on a subclass.

OpenJDK 21 produced `hello component/false/false/true/false/true` before marker
support. The then-current generated Go returned `/false/false/false/false/false`.
The fixture now passes the same Java/Go application parity harness.
