# Labeled non-loop break parity fixture

Java permits `break outer` when `outer` labels an arbitrary statement, including
a block. Go only permits a labeled `break` when the label identifies a `for`,
`switch`, or `select`. The transpiler therefore preserves loop and switch labels
directly, while lowering other labeled statements to a lexical block with a
synthetic end label and rewriting the matching break as a `goto`.

The fixture sends the transfer through `finally`, proving that the generated
control channel runs cleanup before replaying the jump to the block end. Its
byte-exact output is `12`.
