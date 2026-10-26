package stdjava

import "testing"

func TestUUID101LiteralAndFreshStringIdentity(t *testing.T) {
	message := func(input string) (result *JavaString) {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal("UUID input unexpectedly accepted")
			}
			result = JavaThrowableMessageExecution(nil, p)
		}()
		UUIDFromStringJavaString(uuidASCII(input))
		return nil
	}
	large := JavaStringLiteralUTF16([]uint16{'U', 'U', 'I', 'D', ' ', 's', 't', 'r', 'i', 'n', 'g', ' ', 't', 'o', 'o', ' ', 'l', 'a', 'r', 'g', 'e'})
	empty := JavaStringLiteralUTF16(nil)
	if message("00000000-0000-0000-0000-0000000000000") != large || message("1--1-1-1") != empty {
		t.Fatal("literal exception messages must share Java literal identity")
	}
	invalidLiteral := JavaStringLiteralUTF16([]uint16{'I', 'n', 'v', 'a', 'l', 'i', 'd', ' ', 'U', 'U', 'I', 'D', ' ', 's', 't', 'r', 'i', 'n', 'g', ':', ' ', 'x'})
	if message("x") == invalidLiteral {
		t.Fatal("dynamic exception message must remain fresh")
	}
	value := UUIDFromStringJavaString(uuidASCII("1-1-1-1-1"))
	literal := JavaStringLiteralUTF16([]uint16{'0', '0', '0', '0', '0', '0', '0', '1', '-', '0', '0', '0', '1', '-', '0', '0', '0', '1', '-', '0', '0', '0', '1', '-', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '1'})
	first, second := value.StringJava2goExecution(nil), value.StringJava2goExecution(nil)
	if first == literal || first == second {
		t.Fatal("UUID toString results must remain fresh")
	}
}
