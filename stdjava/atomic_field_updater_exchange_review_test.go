package stdjava

import "testing"

// This opaque reference has intentionally conservative function identity. The
// provider bounds an erroneous identity-CAS retry without a timer or leaked
// goroutine, so the V5 RED finishes as an ordinary test failure.
type updaterExchangeReviewState struct {
	calls int
	limit int
	abort any
}
type updaterExchangeReviewFunction func() *updaterExchangeReviewState

func (f updaterExchangeReviewFunction) JavaObjectInfo() *ObjectInfo {
	state := f()
	state.calls++
	if state.calls > state.limit {
		panic(state.abort)
	}
	return nil
}

func TestAtomicReferenceFieldUpdaterGetAndSetOpaqueNonReflexiveIdentity(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
	state := &updaterExchangeReviewState{limit: 16, abort: NewIllegalStateException("exchange identity retry bound")}
	value := updaterExchangeReviewFunction(func() *updaterExchangeReviewState { return state })
	if !ObjectInstanceOf(value, ObjectTypeID) || JavaReferenceEqual(value, value) {
		t.Fatal("regression requires an accepted opaque reference with conservative non-reflexive identity")
	}
	u.Set(target, value)
	state.calls = 0
	replacement := NewObject()
	var old, failure any
	func() {
		defer func() { failure = recover() }()
		old = u.GetAndSet(target, replacement)
	}()
	if failure != nil {
		t.Fatalf("unconditional exchange required identity-CAS retries: provider calls=%d, panic=%T %v", state.calls, failure, failure)
	}
	returned, ok := old.(updaterExchangeReviewFunction)
	if !ok || returned() != state {
		t.Fatal("exchange did not preserve the old opaque callable allocation")
	}
	if state.calls != 1 {
		t.Fatalf("returned view provider calls=%d, want 1", state.calls)
	}
	if !JavaReferenceEqual(u.Get(target), replacement) {
		t.Fatal("exchange did not store the replacement reference")
	}
}

func TestAtomicReferenceFieldUpdaterGetAndSetNativeStringRejectsBeforeMutation(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
	VolatileStore(&target.r, "legacy native string")
	updaterFailure(t, BuiltinThrowableTypeID("UnsupportedOperationException"), func() { u.GetAndSet(target, NewObject()) })
	if VolatileLoad[string](&target.r) != "legacy native string" {
		t.Fatal("unsupported old representation must fail before exchange mutation")
	}
	VolatileStore(&target.r, NullString())
	if old := u.GetAndSet(target, nil); old != nil {
		t.Fatal("native String null sentinel must return Java null")
	}
}
