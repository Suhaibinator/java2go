# Runner provenance

`validate_jvm.py` is a byte-identical copy of the original private runner.
The original runner produced the nine validated JVM observations in
`oracle/oracle.json`. This packaged copy has not been executed and refers to
`/private/tmp` source and output paths; it is archival support, not a portable
ready-to-run gate. See the runner SHA and original command in the oracle.
