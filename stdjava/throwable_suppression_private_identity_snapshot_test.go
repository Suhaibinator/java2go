package stdjava_test

import (
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

// Both methods are legal Java business APIs and deliberately hide the public
// runtime selectors with the signatures produced for Object and Throwable[].
type suppressionIdentityBusiness struct {
	j.Exception
	ticket        any
	prior         j.Throwable
	identityReads int
	historyReads  int
}

func (p *suppressionIdentityBusiness) ThrowableIdentity() any {
	p.identityReads++
	return p.ticket
}

func (p *suppressionIdentityBusiness) SuppressedValues() *j.ReferenceArray {
	p.historyReads++
	return j.SuppressedArray([]any{p.prior})
}

func TestJavaThrowablePrivateStateIgnoresIdentityAndSnapshotBusinessMethods(t *testing.T) {
	causeValue, priorValue := j.NewJavaExceptionMessage(nil), j.NewJavaExceptionMessage(nil)
	cause, prior := &causeValue, &priorValue
	primary := &suppressionIdentityBusiness{Exception: j.NewJavaExceptionMessageCause(nil, cause), ticket: "shared-ticket", prior: prior}
	second := &suppressionIdentityBusiness{Exception: j.NewJavaExceptionMessageCause(nil, cause), ticket: "shared-ticket", prior: prior}
	var alias j.Throwable = primary
	execution := j.NewExecution()
	j.JavaThrowableAddSuppressedExecution(execution, alias, second)
	j.JavaThrowableAddSuppressedExecution(execution, primary, second)
	cleanupValue := j.NewJavaExceptionMessage(nil)
	cleanup := &cleanupValue
	caught := captureSuppressionPanic(func() {
		defer j.CloseResource(func() { panic(cleanup) })
		panic(alias)
	})
	if caught != primary || j.GetCause(primary) != cause {
		t.Error("cleanup changed exact primary or original cause reference")
	}
	checkSnapshot := func(values []any) {
		t.Helper()
		if len(values) != 3 || values[0] != second || values[1] != second || values[2] != cleanup {
			t.Errorf("native snapshot lost ordered duplicate references: %#v", values)
		}
	}
	checkSnapshot(j.GetSuppressed(alias))
	copyOfPrimary := *primary
	checkSnapshot(j.GetSuppressed(&copyOfPrimary))
	self := captureSuppressionPanic(func() {
		j.JavaThrowableAddSuppressedExecution(execution, alias, &copyOfPrimary)
	})
	if !j.CaughtAs(self, "IllegalArgumentException") || j.GetCause(self) != primary {
		t.Errorf("true self suppression was not rejected with original cause: %T", self)
	}
	var absent *suppressionIdentityBusiness
	null := captureSuppressionPanic(func() {
		j.JavaThrowableAddSuppressedExecution(execution, primary, absent)
	})
	if !j.CaughtAs(null, "NullPointerException") {
		t.Errorf("typed-null argument rejection: %T", null)
	}
	nullReceiver := captureSuppressionPanic(func() {
		j.JavaThrowableAddSuppressedExecution(execution, absent, second)
	})
	if !j.CaughtAs(nullReceiver, "NullPointerException") {
		t.Errorf("typed-null receiver rejection: %T", nullReceiver)
	}
	first := j.GetSuppressed(primary)
	if len(first) > 0 {
		first[0] = nil
	}
	checkSnapshot(j.GetSuppressed(primary))
	if primary.identityReads != 0 || primary.historyReads != 0 || second.identityReads != 0 {
		t.Error("native identity or snapshot operation invoked a source business method")
	}
	if primary.ThrowableIdentity() != second.ThrowableIdentity() {
		t.Error("explicit business keys must remain shared")
	}
	business := primary.SuppressedValues()
	if assigned := j.ReferenceArraySet(business, 0, cleanup); assigned != cleanup {
		t.Error("business array assignment lost the exact cleanup reference")
	}
	businessFresh := primary.SuppressedValues()
	if business == businessFresh || j.ReferenceArrayGet[j.Throwable](businessFresh, 0, j.ThrowableTypeID) != prior {
		t.Error("explicit business snapshots lost original history or independence")
	}
	if primary.identityReads != 1 || primary.historyReads != 2 || second.identityReads != 1 {
		t.Error("explicit source business methods did not retain their observable calls")
	}
	checkSnapshot(j.GetSuppressed(primary))
}
