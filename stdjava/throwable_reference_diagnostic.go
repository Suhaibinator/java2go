package stdjava

import (
	"reflect"
	"unicode/utf16"
)

// Select the declared text protocol without invoking Java code. Legacy callback
// registrations take precedence over a canonical detail-message field; the two
// protocols coexist during migration and must not reinterpret each other's ABI.
func throwableHasCanonicalDiagnosticText(value any) bool {
	if javaReferenceIsNull(value) {
		return false
	}
	value = collectionObjectView(value)
	kind := reflect.TypeOf(value)
	if _, ok := javaThrowableToStringOverrides.Load(kind); ok {
		return true
	}
	if _, ok := throwableToStringOverrides.Load(kind); ok {
		return false
	}
	if registeredJavaSourceValue(value) {
		if method, _, found := registeredSourceToStringMethod(value); found {
			return method.IsValid() && method.Type().NumIn() == 1 && method.Type().In(0) == reflect.TypeOf((*Execution)(nil)) && method.Type().NumOut() == 1 && method.Type().Out(0) == reflect.TypeOf((*JavaString)(nil))
		}
	} else {
		if _, ok := value.(javaReferenceStringer); ok {
			return true
		}
		if _, ok := value.(executionStringer); ok {
			return false
		}
	}
	if _, ok := javaThrowableLocalizedMessageOverrides.Load(kind); ok {
		return true
	}
	if _, ok := throwableLocalizedMessageOverrides.Load(kind); ok {
		return false
	}
	if _, ok := javaThrowableMessageOverrides.Load(kind); ok {
		return true
	}
	if _, ok := throwableMessageOverrides.Load(kind); ok {
		return false
	}
	carrier, ok := value.(interface{ throwableCauseState() *throwableState })
	return ok && carrier.throwableCauseState() != nil && carrier.throwableCauseState().canonicalMessage
}

// Called after releasing the state mutex but while retaining initCause's Java
// monitor. The selected cause callback runs exactly once; abrupt completion
// escapes unchanged before any rejection Throwable is allocated.
func newJavaInitCauseRejection(execution *Execution, primary, cause any, referenceText bool) IllegalStateException {
	var description *JavaString
	if javaReferenceIsNull(cause) {
		description = JavaStringLiteralUTF16([]uint16{'a', ' ', 'n', 'u', 'l', 'l'})
	} else if referenceText {
		description = JavaStringValueOfExecution(execution, cause)
	} else {
		// Only the legacy/native protocol enters this one-way diagnostic adapter.
		// Canonical source results above never pass through host encoding/decoding.
		text := StringValueOfExecution(execution, cause)
		if !StringIsNull(text) {
			description = NewJavaStringUTF16(utf16.Encode([]rune(decodeCharset([]byte(text), UTF_8))))
		}
	}
	if description == nil {
		description = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	units := utf16.Encode([]rune("Can't overwrite cause with "))
	units = append(units, description.units...)
	base := newJavaThrowableBase("IllegalStateException", NewJavaStringUTF16(units))
	base.state.cause = primary
	base.state.causeInitialized = true
	return IllegalStateException{base}
}
