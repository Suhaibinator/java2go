# Multiple for-update expressions

Preserved while isolating local identifier hygiene. JDK21 prints `0:1` then `1:2`; generated Go only lowers the first comma-separated update and prints `0:1` then `1:1`. The identifier hygiene regression retains its multi-declarator initializer and moves the second increment into the body to test name binding independently. This original valid Java case remains a separate compiler blocker; no frozen campaign inputs were changed.
