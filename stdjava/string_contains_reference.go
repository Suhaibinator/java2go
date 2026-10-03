package stdjava

// JavaStringContainsExecution implements String.contains(CharSequence) using
// the target's virtual toString and the caller's execution. Search compares
// canonical UTF16 units, including isolated surrogates.
func JavaStringContainsExecution(execution *Execution, text *JavaString, target any) bool {
	RequireJavaString(text)
	ReferenceRequireNonNull(target)
	needle := JavaStringValueOfExecution(execution, target)
	return JavaStringIndexOf(text, RequireJavaString(needle)) >= 0
}
