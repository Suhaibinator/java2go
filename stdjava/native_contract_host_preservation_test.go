package stdjava

import "testing"

type erasedContractHostRightA struct{}
type erasedContractHostRightB struct{}
type erasedContractHostFirst struct{ calls int }

func (v *erasedContractHostFirst) CompareToJava2goExecution(_ *Execution, _ *erasedContractHostRightA) int32 {
	v.calls++
	return 1
}
func (v *erasedContractHostFirst) CompareToJava2goExecution1(_ *Execution, _ *erasedContractHostRightB) int32 {
	v.calls++
	return 2
}

type erasedContractHostSuffix struct {
	want  *Execution
	calls int
}

func (v *erasedContractHostSuffix) CompareToJava2goExecution7(e *Execution, other *erasedContractHostRightA) int32 {
	if e != v.want {
		panic("execution changed")
	}
	v.calls++
	if other == nil {
		return -7
	}
	return 7
}

type erasedContractHostLegacy struct{ calls int }

func (v *erasedContractHostLegacy) CompareTo(other *erasedContractHostRightA) int32 {
	v.calls++
	return 9
}

func TestNativeContractHostFirstShapeRemainsAuthoritative(t *testing.T) {
	left := &erasedContractHostFirst{}
	if result, found := compareViaCompareToExecution(NewExecution(), left, &erasedContractHostRightB{}); found || result != 0 || left.calls != 0 {
		t.Fatalf("retried rejected first host entry: %d %t %d", result, found, left.calls)
	}
	if result, found := compareViaCompareToExecution(NewExecution(), left, &erasedContractHostRightA{}); !found || result != 1 || left.calls != 1 {
		t.Fatalf("first host entry: %d %t %d", result, found, left.calls)
	}
}
func TestNativeContractHostSuffixAndLegacyRemainCallable(t *testing.T) {
	execution := NewExecution()
	suffix := &erasedContractHostSuffix{want: execution}
	if result, found := compareViaCompareToExecution(execution, suffix, &erasedContractHostRightA{}); !found || result != 7 {
		t.Fatalf("suffix entry: %d %t", result, found)
	}
	if result, found := compareViaCompareToExecution(execution, suffix, nil); !found || result != -7 || suffix.calls != 2 {
		t.Fatalf("nullable suffix entry: %d %t %d", result, found, suffix.calls)
	}
	legacy := &erasedContractHostLegacy{}
	if result, found := compareViaCompareToExecution(execution, legacy, &erasedContractHostRightA{}); !found || result != 9 || legacy.calls != 1 {
		t.Fatalf("legacy entry: %d %t %d", result, found, legacy.calls)
	}
}
