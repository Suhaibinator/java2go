# Resource suppression semantics parity

This application exercises the complete Java try-with-resources exceptional
completion protocol against a real JDK oracle. It covers body-primary
exceptions, reverse close and suppression order for multiple resources, a
close-only primary exception, a close failure overriding a pending return,
self-suppression and cause identity, and distinct same-value exception objects.

The checked-in stdout was produced by compiling and running this source with
the JDK. The parity harness requires the generated Go application to match it
byte for byte.
