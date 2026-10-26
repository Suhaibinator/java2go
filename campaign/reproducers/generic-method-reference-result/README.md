# Generic method-reference result counterexample

This imported-member variant of the adjacent scoped-nested-constructor probe passes nested construction, then fails Go compilation because the erased generic method result (`any`) is returned directly from a `Function<String,String>` adapter. JDK21 prints `abc`. The current helper naming regression uses `Function<String,Object>` plus an explicit cast solely to isolate naming; this exact String-result counterexample remains pending.

Use the commands in the adjacent scoped-nested-constructor README, replacing that directory name with `generic-method-reference-result`.
