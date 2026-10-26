package stdjava

// RequireInvocationReceiver implements the invocation-site null boundary after
// generated Java receiver/argument evaluation. The compiler supplies a proven
// source diagnostic; this helper never interprets a Go panic. It intentionally
// makes no source-stack-capture claim in the message-only runtime cohort.
func RequireInvocationReceiver[T any](receiver T, message string) T {
	if javaReferenceIsNull(receiver) {
		panic(NewNullPointerException(message))
	}
	return receiver
}
