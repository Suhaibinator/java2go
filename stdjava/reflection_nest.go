package stdjava

// Descriptor identities come from parsed lexical nesting. A dollar sign in a
// legal top-level Java name never establishes private member access.
func declaredJavaNestMates(left, right TypeID) bool {
	if left == "" || right == "" {
		return false
	}
	host := func(id TypeID) TypeID {
		if declared := classDescriptor(id).NestHost; declared != "" {
			return declared
		}
		return id
	}
	return host(left) == host(right)
}
