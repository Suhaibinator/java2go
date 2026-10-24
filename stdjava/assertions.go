package stdjava

import "os"

// Assertion policy is fixed once at process startup. JAVA2GO_ASSERTIONS=true
// enables all translated source assertions; absent or false disables them.
// Per-class/package and ClassLoader policy changes are not modeled.
var javaAssertionsEnabled = parseJavaAssertionMode(os.Getenv("JAVA2GO_ASSERTIONS"))

func parseJavaAssertionMode(mode string) bool {
	switch mode {
	case "", "false":
		return false
	case "true":
		return true
	default:
		panic("JAVA2GO_ASSERTIONS must be true or false")
	}
}
func JavaAssertionsEnabled() bool { return javaAssertionsEnabled }

// A no-detail assertion has a null message. The Object overload converts its
// detail lazily in generated code and retains a Throwable as its cause.
func NewAssertionFailure(execution *Execution, detail ...any) AssertionError {
	return NewJavaAssertionErrorExecution(execution, detail...)
}
