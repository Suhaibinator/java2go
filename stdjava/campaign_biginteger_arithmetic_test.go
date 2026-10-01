package stdjava

import "testing"

func TestCampaignBigIntegerArithmeticNative(t *testing.T) {
	a, b := NewBigInteger("18446744073709551617"), NewBigInteger("-4294967297")
	product := a.Multiply(b)
	a.Add(b)
	a.Subtract(b)
	product.Divide(BigIntegerValueOf(11))
	product.Mod(BigIntegerValueOf(97))
	if a.String() != "18446744073709551617" || b.String() != "-4294967297" {
		t.Fatal("BigInteger operand mutated")
	}
	for _, row := range []struct {
		text string
		bits int32
	}{{"0", 0}, {"-1", 0}, {"-2", 1}, {"-3", 2}, {"-2147483648", 31}, {"2147483648", 32}, {"-18446744073709551616", 64}, {"-18446744073709551617", 65}} {
		if got := NewBigInteger(row.text).BitLength(); got != row.bits {
			t.Fatalf("%s bitLength=%d want %d", row.text, got, row.bits)
		}
	}
	if got := NewBigInteger("-7").Divide(BigIntegerValueOf(3)).String(); got != "-2" {
		t.Fatalf("division %s", got)
	}
	if got := NewBigInteger("-7").Mod(BigIntegerValueOf(3)).String(); got != "2" {
		t.Fatalf("modulo %s", got)
	}
	for _, row := range []struct {
		name, kind, message string
		call                func()
	}{{"zero divisor", "ArithmeticException", "BigInteger divide by zero", func() { a.Divide(BigIntegerValueOf(0)) }}, {"zero modulus", "ArithmeticException", "BigInteger: modulus not positive", func() { a.Mod(BigIntegerValueOf(0)) }}, {"negative modulus", "ArithmeticException", "BigInteger: modulus not positive", func() { a.Mod(BigIntegerValueOf(-1)) }}, {"null rhs", "NullPointerException", "", func() { a.Add(nil) }}, {"null receiver", "NullPointerException", "", func() { var missing *BigInteger; missing.Multiply(b) }}} {
		t.Run(row.name, func(t *testing.T) {
			defer func() {
				failure := recover()
				if failure == nil || !CaughtAs(failure, row.kind) {
					t.Fatalf("failure=%v want %s", failure, row.kind)
				}
				if row.message != "" {
					if GetMessage(failure) != row.message {
						t.Fatalf("message %q want %q", GetMessage(failure), row.message)
					}
				}
			}()
			row.call()
		})
	}
}
