# Try-with-resources suppressed-exception parity

When the resource body and `close()` both throw, Java keeps the body exception
as the primary completion and attaches the close exception as suppressed.
Generated cleanup now captures the pending body exception, runs `close()` under
a separate recovery boundary, records any close failure on the primary
throwable, and then replays the primary exception for catch dispatch.

Focused integration coverage also verifies multiple resources, reverse close
order, suppression order, a close-only primary exception, and a close failure
superseding a pending return.
