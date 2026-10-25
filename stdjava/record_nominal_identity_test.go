package stdjava

import "testing"

type nominalRecordControl struct{}

func (*nominalRecordControl) JavaDynamicTypeID() TypeID { return "tests.RecordOwner$Entry" }

func TestRecordNominalParentAndStrictViews(t *testing.T) {
	const entry TypeID = "tests.RecordOwner$Entry"
	RegisterJavaType(entry, RecordTypeID)
	for _, target := range []TypeID{entry, RecordTypeID, ObjectTypeID} {
		if !JavaTypeAssignable(entry, target) {
			t.Fatalf("%s must be assignable to %s", entry, target)
		}
	}
	value := &nominalRecordControl{}
	if ObjectView[*nominalRecordControl](value, entry) != value {
		t.Fatal("record view lost allocation identity")
	}
	if ObjectView[any](value, RecordTypeID) != value {
		t.Fatal("Record parent view lost allocation identity")
	}
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		ObjectView[*nominalRecordControl](value, "tests.OtherRecord")
	}()
	if !CaughtAs(recovered, "ClassCastException") {
		t.Fatalf("unrelated record view panic=%T (%v)", recovered, recovered)
	}
}
