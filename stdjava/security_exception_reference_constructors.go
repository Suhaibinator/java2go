package stdjava

// GeneralSecurityException is the checked JDK superclass of algorithm failures.
// Each value embeds the same shared Java Throwable identity and state carrier.
type GeneralSecurityException struct{ ThrowableBase }

// NewJavaGeneralSecurityExceptionMessage preserves the String/no-argument constructor's
// exact nullable detailMessage and its initially available cause slot.
func NewJavaGeneralSecurityExceptionMessage(message *JavaString) GeneralSecurityException {
	return GeneralSecurityException{newJavaThrowableBase("GeneralSecurityException", message)}
}

// NewJavaGeneralSecurityExceptionCauseExecution invokes the cause's virtual toString in the
// original caller execution. An abrupt callback propagates before allocation.
func NewJavaGeneralSecurityExceptionCauseExecution(execution *Execution, cause any) GeneralSecurityException {
	return GeneralSecurityException{newJavaThrowableCause(execution, "GeneralSecurityException", cause)}
}

func NewJavaGeneralSecurityExceptionMessageCause(message *JavaString, cause any) GeneralSecurityException {
	base := newJavaThrowableBase("GeneralSecurityException", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return GeneralSecurityException{base}
}

// NewJavaNoSuchAlgorithmExceptionMessage preserves the String/no-argument constructor's
// exact nullable detailMessage and its initially available cause slot.
func NewJavaNoSuchAlgorithmExceptionMessage(message *JavaString) NoSuchAlgorithmException {
	return NoSuchAlgorithmException{newJavaThrowableBase("NoSuchAlgorithmException", message)}
}

// NewJavaNoSuchAlgorithmExceptionCauseExecution invokes the cause's virtual toString in the
// original caller execution. An abrupt callback propagates before allocation.
func NewJavaNoSuchAlgorithmExceptionCauseExecution(execution *Execution, cause any) NoSuchAlgorithmException {
	return NoSuchAlgorithmException{newJavaThrowableCause(execution, "NoSuchAlgorithmException", cause)}
}

func NewJavaNoSuchAlgorithmExceptionMessageCause(message *JavaString, cause any) NoSuchAlgorithmException {
	base := newJavaThrowableBase("NoSuchAlgorithmException", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return NoSuchAlgorithmException{base}
}
