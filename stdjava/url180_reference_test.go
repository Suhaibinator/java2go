package stdjava

import (
	"reflect"
	"testing"
)

func TestURL180ExistingNominalMetadata(t *testing.T) {
	t.Run("URLSerializable", func(t *testing.T) {
		if !JavaTypeAssignable(TypeID("java.net.URL"), SerializableTypeID) {
			t.Fatal("canonical URL must implement java.io.Serializable")
		}
	})
	t.Run("MalformedName", func(t *testing.T) {
		if got := BuiltinThrowableTypeID("MalformedURLException"); got != TypeID("java.net.MalformedURLException") {
			t.Fatalf("canonical checked exception identity=%q", got)
		}
	})
	t.Run("MalformedSuperclass", func(t *testing.T) {
		parent := ClassLiteral(TypeID("java.net.MalformedURLException")).GetSuperclass()
		if parent == nil || parent.TypeID() != TypeID("java.io.IOException") {
			t.Fatalf("canonical checked exception superclass=%v", parent)
		}
	})
}

func url180Recovered(action func()) (failure any) {
	defer func() { failure = recover() }()
	action()
	return nil
}

func TestURL180ReferenceUTF16AndCauseState(t *testing.T) {
	url180ProviderRegistryScope(t)
	execution := NewExecution()
	units := []uint16{'h', 't', 't', 'p', ':', '/', '/', 'h', '/', 'x', 0xd800, '?', 0xdfff, '#', 0xd801}
	original := NewJavaStringUTF16(units)
	value := NewJavaURLStringExecution(execution, original)
	units[0] = 'X'
	if got := value.ToExternalFormJava2goExecution(execution).UTF16Copy(); !reflect.DeepEqual(got, original.UTF16Copy()) {
		t.Fatalf("UTF16 changed: %x", got)
	}
	alias := value
	if !JavaReferenceEqual(value, alias) || JavaReferenceEqual(value, NewJavaURLStringExecution(execution, original)) {
		t.Fatal("allocation/reference identity lost")
	}
	if ClassLiteral(value.JavaDynamicTypeID()).TypeID() != TypeID("java.net.URL") {
		t.Fatal("dynamic class identity")
	}
	for _, action := range []func(){func() { ObjectEqualsExecution(execution, value, alias) }, func() { ObjectHashCodeExecution(execution, value) }} {
		if failure := url180Recovered(action); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("erased unsupported service=%T %v", failure, failure)
		}
	}
	var nilURL *JavaURL
	if failure := url180Recovered(func() { nilURL.ToExternalFormJava2goExecution(execution) }); !CaughtAs(failure, "NullPointerException") {
		t.Fatalf("null receiver=%T", failure)
	}
	for _, protocol := range []string{"ftp", "jrt", "jmod", "mailto"} {
		if failure := url180Recovered(func() { NewJavaURLStringExecution(execution, uriRuntimeText(protocol+":value")) }); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("known unsupported handler disguised: %s %T", protocol, failure)
		}
	}
	message := NewJavaStringUTF16([]uint16{'m', 0xd800})
	failure := NewJavaMalformedURLExceptionMessage(message)
	if JavaThrowableMessageExecution(execution, failure) != message || failure.state.causeInitialized {
		t.Fatal("message identity/cause slot changed")
	}
	_ = ThrowableInitCauseDefaultExecution(execution, failure, nil)
	if !failure.state.causeInitialized {
		t.Fatal("cause initialization missing")
	}
	typedNil := NewJavaMalformedURLExceptionMessage(nil)
	if JavaThrowableMessageExecution(execution, typedNil) != nil || typedNil.state.causeInitialized {
		t.Fatal("typed null message state")
	}
	for _, input := range []string{"relative", "unknown-service:x"} {
		f := url180Recovered(func() { NewJavaURLStringExecution(execution, uriRuntimeText(input)) })
		m, ok := f.(MalformedURLException)
		if !ok || m.state.causeInitialized || m.CauseValue() != nil {
			t.Fatalf("message-only malformed state=%T", f)
		}
	}
	cause := NewJavaIllegalArgumentExceptionMessage(message)
	wrapped := urlMalformedFromCauseExecution(execution, cause)
	if JavaThrowableMessageExecution(execution, wrapped) != message || !JavaReferenceEqual(wrapped.CauseValue(), cause) || !wrapped.state.causeInitialized {
		t.Fatal("wrapped parser cause/message identity")
	}
	for _, parent := range []string{"MalformedURLException", "IOException", "Exception", "Throwable"} {
		if !CaughtAs(wrapped, parent) {
			t.Fatalf("missing catch parent %s", parent)
		}
	}
	if !JavaTypeAssignable(wrapped.JavaDynamicTypeID(), SerializableTypeID) || JavaTypeAssignable(value.JavaDynamicTypeID(), ComparableTypeID) {
		t.Fatal("nominal hierarchy wrong")
	}
}

// These handlers exist in the pinned JDK module image/default factory. This
// native service does not implement them yet: preserve an explicit positive
// compatibility gap instead of misreporting an unknown/malformed protocol.
func TestURL701ValidUnimplementedHandlerGaps(t *testing.T) {
	url180ProviderRegistryScope(t)
	execution := NewExecution()
	for _, entry := range []struct{ protocol, spec string }{
		{"ftp", "ftp://example.invalid/a"},
		{"jrt", "jrt:/java.base/java/lang/Object.class"},
		{"jmod", "jmod:/java.base"},
		{"mailto", "mailto:person@example.invalid"},
	} {
		t.Run(entry.protocol, func(t *testing.T) {
			failure := url180Recovered(func() { NewJavaURLStringExecution(execution, JavaStringFromHostUTF8(entry.spec)) })
			if !CaughtAs(failure, "UnsupportedOperationException") || CaughtAs(failure, "MalformedURLException") {
				t.Fatalf("valid unimplemented handler must remain explicit: %s %T %v", entry.protocol, failure, failure)
			}
			want := "java.net.URL default handler " + entry.protocol + " is not implemented"
			if got := JavaThrowableMessageExecution(execution, failure); got == nil || !reflect.DeepEqual(got.UTF16Copy(), JavaStringFromHostUTF8(want).UTF16Copy()) {
				t.Fatalf("handler gap message %v, want %q", got, want)
			}
		})
	}
}

// These constructor boundaries follow the pinned JDK numeric IPv6 predicate.
// Unknown panics and malformed error state never count as intended semantic RED.
func TestURL701IPv6NumericBoundaries(t *testing.T) {
	url180ProviderRegistryScope(t)
	execution := NewExecution()
	for _, entry := range []struct {
		name, spec string
		malformed  bool
	}{
		{"hex_four", "http://[0000::1]/", false},
		{"hex_leading_zero_five", "http://[00000::1]/", false},
		{"hex_overflow", "http://[10000::1]/", true},
		{"embedded_ipv4_length15", "http://[::ffff:000000000.1.1.1]/", false},
		{"embedded_ipv4_length16", "http://[::ffff:0000000000.1.1.1]/", true},
	} {
		t.Run(entry.name, func(t *testing.T) {
			var value *JavaURL
			failure := url180Recovered(func() { value = NewJavaURLStringExecution(execution, JavaStringFromHostUTF8(entry.spec)) })
			requireExactMalformedHost := func() {
				t.Helper()
				malformed, ok := failure.(MalformedURLException)
				if !ok || value != nil || !malformed.state.causeInitialized {
					t.Fatalf("unexpected constructor failure class or state: %T", failure)
				}
				cause, ok := malformed.CauseValue().(IllegalArgumentException)
				if !ok || cause.state.causeInitialized || cause.CauseValue() != nil {
					t.Fatalf("unexpected malformed host cause class or state: %T", malformed.CauseValue())
				}
				want := JavaStringFromHostUTF8("Invalid host: " + entry.spec[7:len(entry.spec)-1]).UTF16Copy()
				for _, throwable := range []any{malformed, cause} {
					message := JavaThrowableMessageExecution(execution, throwable)
					if message == nil || !reflect.DeepEqual(message.UTF16Copy(), want) {
						t.Fatalf("unexpected malformed host message: %T", throwable)
					}
				}
			}
			requireExactSuccessfulURL := func() {
				t.Helper()
				if failure != nil || value == nil {
					t.Fatalf("unexpected unsuccessful URL result: %T", failure)
				}
				if !reflect.DeepEqual(value.ToExternalFormJava2goExecution(execution).UTF16Copy(), JavaStringFromHostUTF8(entry.spec).UTF16Copy()) {
					t.Fatal("unexpected URL external form")
				}
			}
			if entry.malformed {
				if failure == nil {
					requireExactSuccessfulURL()
					t.Error("JDK-malformed numeric host was admitted")
				} else {
					requireExactMalformedHost()
				}
				return
			}
			if failure != nil {
				requireExactMalformedHost()
				t.Error("JDK-valid numeric host was rejected")
				return
			}
			requireExactSuccessfulURL()
		})
	}
}
