package stdjava

import "testing"

func inetRecovered(action func()) (failure any) {
	defer func() { failure = recover() }()
	action()
	return nil
}
func TestInetAddressJDK21Numeric(t *testing.T) {
	execution := NewExecution()
	rows := []struct {
		input, output, kind string
		hash                int32
	}{
		{"127.0.0.1", "127.0.0.1", "Inet4Address", 2130706433}, {"127.1", "127.0.0.1", "Inet4Address", 2130706433}, {"2130706433", "127.0.0.1", "Inet4Address", 2130706433}, {"127.0.1", "127.0.0.1", "Inet4Address", 2130706433}, {"001.002.003.004", "1.2.3.4", "Inet4Address", 16909060},
		{"::1", "0:0:0:0:0:0:0:1", "Inet6Address", 1}, {"[::1]", "0:0:0:0:0:0:0:1", "Inet6Address", 1}, {"2001:db8::1", "2001:db8:0:0:0:0:0:1", "Inet6Address", 536939705},
		{"::ffff:192.0.2.1", "192.0.2.1", "Inet4Address", -1073741311}, {"::ffff:c000:201", "192.0.2.1", "Inet4Address", -1073741311}, {"fe80::1%2", "fe80:0:0:0:0:0:0:1%2", "Inet6Address", -41943039}, {"::1%0", "0:0:0:0:0:0:0:1%0", "Inet6Address", 1},
	}
	for _, row := range rows {
		t.Run(row.input, func(t *testing.T) {
			a := JavaInetAddressGetByNameExecution(execution, uriRuntimeText(row.input))
			b := JavaInetAddressGetByNameExecution(execution, uriRuntimeText(row.input))
			if !a.GetHostAddressJava2goExecution(execution).Equals(uriRuntimeText(row.output)) {
				t.Fatalf("address=%v", a.GetHostAddressJava2goExecution(execution))
			}
			if a.JavaDynamicTypeID() != TypeID("java.net."+row.kind) {
				t.Fatalf("class=%v", a.JavaDynamicTypeID())
			}
			if !ObjectEqualsExecution(execution, a, b) || JavaReferenceEqual(a, b) || ObjectHashCodeExecution(execution, a) != row.hash {
				t.Fatal("equals/identity/hash differs")
			}
			if !a.StringJava2goExecution(execution).Equals(uriRuntimeText("/" + row.output)) {
				t.Fatal("toString differs")
			}
			if !JavaTypeAssignable(a.JavaDynamicTypeID(), "java.net.InetAddress") || !JavaTypeAssignable(a.JavaDynamicTypeID(), SerializableTypeID) {
				t.Fatal("ancestry differs")
			}
		})
	}
	a := JavaInetAddressGetByNameExecution(execution, nil)
	b := JavaInetAddressGetByNameExecution(execution, uriRuntimeText(""))
	if !JavaReferenceEqual(a, b) || !a.StringJava2goExecution(execution).Equals(uriRuntimeText("localhost/127.0.0.1")) {
		t.Fatal("cached loopback differs")
	}
}
func TestInetAddressJDK21FailuresAndException(t *testing.T) {
	execution := NewExecution()
	for _, input := range []string{"::1%", "[127.0.0.1]", "[::g]", "[::1", "::g", "1::2::3", "::g%eth0"} {
		failure := inetRecovered(func() { JavaInetAddressGetByNameExecution(execution, uriRuntimeText(input)) })
		if !CaughtAs(failure, "UnknownHostException") || !CaughtAs(failure, "IOException") || !JavaThrowableMessageExecution(execution, failure).Equals(uriRuntimeText(input+": invalid IPv6 address literal")) || GetCause(failure) != nil {
			t.Fatalf("%s: %T %v", input, failure, failure)
		}
	}
	failure := inetRecovered(func() { JavaInetAddressGetByNameExecution(execution, uriRuntimeText("0x7f000001")) })
	if !CaughtAs(failure, "UnknownHostException") || !CaughtAs(GetCause(failure), "IllegalArgumentException") || !JavaThrowableMessageExecution(execution, GetCause(failure)).Equals(uriRuntimeText("Invalid IP address literal: 0x7f000001")) {
		t.Fatalf("ambiguous cause=%v", failure)
	}
	bad := NewJavaStringUTF16([]uint16{':', ':', 'g', 0xd800})
	badFailure := inetRecovered(func() { JavaInetAddressGetByNameExecution(execution, bad) })
	if !CaughtAs(badFailure, "UnknownHostException") || !JavaThrowableMessageExecution(execution, badFailure).Equals(javaInetDiagnostic(bad, ": invalid IPv6 address literal")) {
		t.Fatal("UTF16 malformed diagnostic")
	}
	nul := NewJavaStringUTF16([]uint16{'1', 0, 'x'})
	failure = inetRecovered(func() { JavaInetAddressGetByNameExecution(execution, nul) })
	if !CaughtAs(failure, "UnknownHostException") || !JavaThrowableMessageExecution(execution, failure).Equals(uriRuntimeText("NUL character not allowed in hostname")) {
		t.Fatal("NUL contract")
	}
	message := NewJavaStringUTF16([]uint16{'m', 0xd800})
	e := NewJavaUnknownHostExceptionMessage(message)
	if JavaThrowableMessageExecution(execution, e) != message || GetCause(e) != nil || e.JavaDynamicTypeID() != TypeID("java.net.UnknownHostException") || ClassLiteral(e.JavaDynamicTypeID()).GetSuperclass().TypeID() != TypeID("java.io.IOException") {
		t.Fatal("exception metadata")
	}
	cause := NewJavaIOExceptionMessage(uriRuntimeText("cause"))
	if ThrowableInitCauseDefaultExecution(execution, e, cause).(UnknownHostException).state != e.state || !JavaReferenceEqual(GetCause(e), cause) {
		t.Fatal("writable cause identity")
	}
	if JavaThrowableMessageExecution(execution, NewJavaUnknownHostExceptionMessage(nil)) != nil {
		t.Fatal("null message")
	}
	for _, input := range []string{"fe80::1%eth0"} {
		if f := inetRecovered(func() { JavaInetAddressGetByNameExecution(execution, uriRuntimeText(input)) }); !CaughtAs(f, "UnsupportedOperationException") {
			t.Fatalf("unsupported resolver service %s: %v", input, f)
		}
	}
	localhost := JavaInetAddressGetByNameExecution(execution, uriRuntimeText("localhost"))
	if !localhost.GetHostAddressJava2goExecution(execution).Equals(uriRuntimeText("127.0.0.1")) {
		t.Fatal("OS localhost resolver")
	}
	if JavaInetAddressGetByNameExecution(execution, uriRuntimeText("localhost")) != localhost {
		t.Fatal("positive cache identity")
	}
	var value *JavaInetAddress
	if f := inetRecovered(func() { value.GetHostAddressJava2goExecution(execution) }); !CaughtAs(f, "NullPointerException") {
		t.Fatal("null receiver")
	}
}
