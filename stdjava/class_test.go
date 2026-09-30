package stdjava

import "testing"

func TestClassLiteralIsCanonicalAndCarriesTypeIdentity(t *testing.T) {
	first := ClassLiteral(DoubleTypeID)
	second := ClassLiteral(DoubleTypeID)
	other := ClassLiteral(IntegerTypeID)
	if first != second {
		t.Fatal("repeated ClassLiteral(DoubleTypeID) did not return the canonical object")
	}
	if first == other {
		t.Fatal("different class literals returned the same object")
	}
	if got := first.TypeID(); got != DoubleTypeID {
		t.Fatalf("Class literal TypeID = %q, want %q", got, DoubleTypeID)
	}
}
