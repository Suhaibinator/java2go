# Sol checkpoint 232: Avoid irrelevant ArrayList source proofs during import resolution

Source ArrayList recognition audited whole import namespaces before rejecting definitely unrelated superclass spellings. A necessary-only spelling check now rejects those cases before the existing ownership and native-contract proof. Potential ArrayList names, comments, escapes and unknown syntax still take the complete proof; inherited source walks remain unchanged.

The unchanged import205 regression improves from small=9525/large=50546 to small=8290/large=14767 against its unchanged formula (new ceiling=24708). All three samples pass in normal and race modes. Seventy-four original/focused stages pass, including namespace, mutation, lifetime, constructor and override controls. Independent actual review: cd118742ae30e85402e8db6aaacb135465b03ca66cfa4b784f74ccc280f9ef61.

The existing parser does not translate all Java Unicode identifier escapes; the conservative check preserves the original full-predicate result rather than claiming support. Full Netty strict conversion and its separate race query timeout remain active blockers.
