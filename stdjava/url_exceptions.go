package stdjava

// MalformedURLException is a checked IOException with the same canonical
// message/cause state as other runtime-owned Java throwables.
type MalformedURLException struct{ ThrowableBase }

func NewJavaMalformedURLExceptionMessage(message *JavaString) MalformedURLException {
	return MalformedURLException{newJavaThrowableBase("MalformedURLException", message)}
}

func urlMalformedFromCauseExecution(execution *Execution, cause any) MalformedURLException {
	failure := NewJavaMalformedURLExceptionMessage(JavaThrowableMessageExecution(execution, cause))
	_ = ThrowableInitCauseDefaultExecution(execution, failure, cause)
	return failure
}

func init() { RegisterException("MalformedURLException", "IOException") }
