# Try-with-resources suppressed-exception known gap

When the resource body and `close()` both throw, Java keeps the body exception
as the primary completion and attaches the close exception as suppressed. A Go
defer that panics while another panic is unwinding replaces the first panic, so
the current lowering dispatches the close exception instead.

This fixture remains `known_gap` until resource cleanup explicitly preserves the
primary exception and records secondary close failures. It is intentionally
separate from loop-control parity.
