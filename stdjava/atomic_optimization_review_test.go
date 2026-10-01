package stdjava

import (
	"testing"
	"time"
)

// These are native runtime diagnostics. They do not establish generated Java
// program correctness or JVM-versus-Go performance. No implementation stamp
// fields are inspected by the behavior controls.
const optimizationOwner TypeID = "optimization.fixture.Owner"

type optimizationTarget struct {
	i, l, r, other, shadow VolatileFieldCell
	want                   *Execution
	onResolve              func(*Execution, string)
	info                   *ObjectInfo
}

func (p *optimizationTarget) JavaDynamicTypeID() TypeID {
	if p.info != nil {
		return p.info.DynamicType()
	}
	return optimizationOwner
}
func (p *optimizationTarget) JavaObjectInfo() *ObjectInfo { return p.info }
func (p *optimizationTarget) Java2goOptimizationCellExecution(e *Execution) *VolatileFieldCell {
	optimizationCheck(e != nil && (p.want == nil || p.want == e), "accessor execution")
	return &p.r
}
func optimizationCheck(ok bool, message string) {
	if !ok {
		panic(NewIllegalStateException(message))
	}
}
func optimizationFixture() (*optimizationTarget, *Execution, *AtomicReferenceFieldUpdater) {
	RegisterJavaType(optimizationOwner, ObjectTypeID)
	resolve := func(e *Execution, v any, name string) *VolatileFieldCell {
		p := v.(*optimizationTarget)
		optimizationCheck(e != nil && (p.want == nil || p.want == e), "resolver execution")
		if p.onResolve != nil {
			p.onResolve(e, name)
		}
		switch name {
		case "i":
			return &p.i
		case "l":
			return &p.l
		case "r":
			return &p.r
		default:
			return &p.other
		}
	}
	RegisterClassDescriptor(ClassDescriptor{Type: optimizationOwner, HasModifiers: true, Modifiers: 1, Fields: []FieldDescriptor{
		{Name: "i", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(e *Execution, v any) *VolatileFieldCell { return resolve(e, v, "i") }},
		{Name: "l", Type: PrimitiveLongTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(e *Execution, v any) *VolatileFieldCell { return resolve(e, v, "l") }},
		{Name: "r", Type: ObjectTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(e *Execution, v any) *VolatileFieldCell { return resolve(e, v, "r") },
			Get: func(e *Execution, v any) any { return VolatileLoad[any](resolve(e, v, "r")) },
			Set: func(e *Execution, v, x any) { VolatileStore(resolve(e, v, "r"), x) }},
	}})
	e := NewExecution()
	p := &optimizationTarget{want: e}
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(optimizationOwner), ClassLiteral(ObjectTypeID), "r", optimizationOwner)
	return p, e, u
}

type optimizationResult struct{ value, failure any }

func optimizationComplete(t *testing.T, action func() any) any {
	t.Helper()
	done := make(chan optimizationResult, 1)
	go func() {
		r := optimizationResult{}
		defer func() { r.failure = recover(); done <- r }()
		r.value = action()
	}()
	select {
	case r := <-done:
		if r.failure != nil {
			t.Fatalf("bounded operation panic: %T", r.failure)
		}
		return r.value
	case <-time.After(2 * time.Second):
		t.Fatal("bounded operation did not terminate; owned package must fail")
		return nil
	}
}
func optimizationFailure(action func()) (failure any) {
	defer func() { failure = recover() }()
	action()
	return nil
}

type optimizationAPI struct {
	name       string
	returnsNew bool
}

var optimizationAPIs = []optimizationAPI{{"GetAndUpdate", false}, {"UpdateAndGet", true}, {"GetAndAccumulate", false}, {"AccumulateAndGet", true}}

func (api optimizationAPI) invoke(u *AtomicReferenceFieldUpdater, e *Execution, p any, operand any, f func(*Execution, any) any) any {
	switch api.name {
	case "GetAndUpdate":
		return u.GetAndUpdateExecution(e, p, f)
	case "UpdateAndGet":
		return u.UpdateAndGetExecution(e, p, f)
	case "GetAndAccumulate":
		return u.GetAndAccumulateExecution(e, p, operand, func(actual *Execution, old, x any) any {
			optimizationCheck(x == operand, "operand alias")
			return f(actual, old)
		})
	default:
		return u.AccumulateAndGetExecution(e, p, operand, func(actual *Execution, old, x any) any {
			optimizationCheck(x == operand, "operand alias")
			return f(actual, old)
		})
	}
}
func optimizationResultIs(api optimizationAPI, result, old, next any) bool {
	if api.returnsNew {
		return JavaReferenceEqual(result, next)
	}
	return JavaReferenceEqual(result, old)
}

type optimizationWriter struct {
	name  string
	write func(*optimizationTarget, *Execution, *AtomicReferenceFieldUpdater, any)
}

var optimizationWriters = []optimizationWriter{
	{"set", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		u.SetExecution(e, p, x)
	}},
	{"lazy_set", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		u.LazySetExecution(e, p, x)
	}},
	{"volatile_store", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		VolatileStore(&p.r, x)
	}},
	{"reflection_set", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		ClassLiteral(optimizationOwner).GetDeclaredField("r").SetExecution(e, p, x)
	}},
	{"cas", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		optimizationCheck(u.CompareAndSetExecution(e, p, VolatileLoad[any](&p.r), x), "writer CAS")
	}},
	{"weak_cas", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		optimizationCheck(u.WeakCompareAndSetExecution(e, p, VolatileLoad[any](&p.r), x), "writer weak CAS")
	}},
	{"exchange", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		u.GetAndSetExecution(e, p, x)
	}},
	{"nested_callback", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		u.UpdateAndGetExecution(e, p, func(actual *Execution, _ any) any { optimizationCheck(actual == e, "nested execution"); return x })
	}},
	// This exercises the generated-accessor ABI, not an autogenerated program.
	{"accessor_store", func(p *optimizationTarget, e *Execution, u *AtomicReferenceFieldUpdater, x any) {
		VolatileStore(ReflectGeneratedVolatileFieldCellExecution(e, p, optimizationOwner, "Java2goOptimizationCellExecution"), x)
	}},
}

func TestAtomicOptimizationBehavior01DefaultUnobservedCapture(t *testing.T) {
	for _, api := range optimizationAPIs {
		t.Run(api.name, func(t *testing.T) {
			p, e, u := optimizationFixture()
			next, operand := NewObject(), NewObject()
			calls := 0
			result := optimizationComplete(t, func() any {
				return api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
					calls++
					optimizationCheck(calls == 1 && actual == e && nilJavaReference(old), "default callback")
					return next
				})
			})
			if calls != 1 || !optimizationResultIs(api, result, nil, next) || !JavaReferenceEqual(VolatileLoad[any](&p.r), next) {
				t.Fatal("default capture/result")
			}
		})
	}
}

func TestAtomicOptimizationBehavior02SameValueWrite(t *testing.T) {
	for _, api := range optimizationAPIs {
		for _, writer := range optimizationWriters {
			for _, empty := range []bool{false, true} {
				name := api.name + "/" + writer.name + "/object"
				if empty {
					name = api.name + "/" + writer.name + "/default_null"
				}
				t.Run(name, func(t *testing.T) {
					p, e, u := optimizationFixture()
					var old any
					if !empty {
						old = NewObject()
						VolatileStore(&p.r, old)
					}
					next, operand := NewObject(), NewObject()
					calls := 0
					result := optimizationComplete(t, func() any {
						return api.invoke(u, e, p, operand, func(actual *Execution, value any) any {
							calls++
							optimizationCheck(calls <= 2 && actual == e && JavaReferenceEqual(value, old), "same-value callback")
							if calls == 1 {
								writer.write(p, e, u, old)
							}
							return next
						})
					})
					if calls != 2 || !optimizationResultIs(api, result, old, next) || !JavaReferenceEqual(VolatileLoad[any](&p.r), next) {
						t.Fatal("same-value write was missed")
					}
				})
			}
		}
	}
}

func TestAtomicOptimizationBehavior03ABAWrite(t *testing.T) {
	for _, api := range optimizationAPIs {
		for _, writer := range optimizationWriters {
			t.Run(api.name+"/"+writer.name, func(t *testing.T) {
				p, e, u := optimizationFixture()
				a, b, next, operand := NewObject(), NewObject(), NewObject(), NewObject()
				VolatileStore(&p.r, a)
				calls := 0
				result := optimizationComplete(t, func() any {
					return api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
						calls++
						optimizationCheck(calls <= 2 && actual == e && JavaReferenceEqual(old, a), "ABA callback")
						if calls == 1 {
							writer.write(p, e, u, b)
							writer.write(p, e, u, a)
						}
						return next
					})
				})
				if calls != 2 || !optimizationResultIs(api, result, a, next) || !JavaReferenceEqual(VolatileLoad[any](&p.r), next) {
					t.Fatal("ABA write was missed")
				}
			})
		}
	}
}

func TestAtomicOptimizationBehavior04MultipleObservers(t *testing.T) {
	for _, api := range optimizationAPIs {
		t.Run(api.name, func(t *testing.T) {
			p, e, u := optimizationFixture()
			a, b, c, operand := NewObject(), NewObject(), NewObject(), NewObject()
			VolatileStore(&p.r, a)
			optimizationComplete(t, func() any {
				entered := make(chan struct{})
				release := make(chan struct{})
				done := make(chan optimizationResult, 1)
				calls := 0
				go func() {
					r := optimizationResult{}
					defer func() { r.failure = recover(); done <- r }()
					r.value = api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
						calls++
						optimizationCheck(calls <= 2 && actual == e, "observer callback")
						if calls == 1 {
							optimizationCheck(JavaReferenceEqual(old, a), "observer first")
							close(entered)
							<-release
						} else {
							optimizationCheck(JavaReferenceEqual(old, b), "observer retry")
						}
						return c
					})
				}()
				<-entered
				first := api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
					optimizationCheck(actual == e && JavaReferenceEqual(old, a), "committing observer")
					return b
				})
				optimizationCheck(optimizationResultIs(api, first, a, b), "first observer result")
				close(release)
				r := <-done
				if r.failure != nil {
					panic(r.failure)
				}
				optimizationCheck(calls == 2 && optimizationResultIs(api, r.value, b, c) && JavaReferenceEqual(VolatileLoad[any](&p.r), c), "observer commit order")
				return nil
			})
		})
	}
}

func TestAtomicOptimizationBehavior05WritesWithoutIntermediateCapture(t *testing.T) {
	for _, api := range optimizationAPIs {
		t.Run(api.name, func(t *testing.T) {
			p, e, u := optimizationFixture()
			a, b, c, next, operand := NewObject(), NewObject(), NewObject(), NewObject(), NewObject()
			VolatileStore(&p.r, a)
			calls := 0
			result := optimizationComplete(t, func() any {
				return api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
					calls++
					optimizationCheck(calls <= 2 && actual == e, "write sequence callback")
					if calls == 1 {
						optimizationCheck(JavaReferenceEqual(old, a), "sequence old")
						VolatileStore(&p.r, b)
						u.SetExecution(e, p, a)
						VolatileStore(&p.r, c)
					} else {
						optimizationCheck(JavaReferenceEqual(old, c), "sequence latest")
					}
					return next
				})
			})
			if calls != 2 || !optimizationResultIs(api, result, c, next) || !JavaReferenceEqual(VolatileLoad[any](&p.r), next) {
				t.Fatal("write-only sequence missed")
			}
		})
	}
}

func TestAtomicOptimizationBehavior06NewObserverAfterInvalidation(t *testing.T) {
	for _, api := range optimizationAPIs {
		t.Run(api.name, func(t *testing.T) {
			p, e, u := optimizationFixture()
			a, b, next, operand := NewObject(), NewObject(), NewObject(), NewObject()
			VolatileStore(&p.r, a)
			calls := 0
			result := optimizationComplete(t, func() any {
				return api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
					calls++
					optimizationCheck(calls <= 2 && actual == e, "outer execution")
					if calls == 1 {
						optimizationCheck(JavaReferenceEqual(old, a), "outer first")
						u.SetExecution(e, p, a)
						inner := u.GetAndUpdateExecution(e, p, func(innerE *Execution, innerOld any) any {
							optimizationCheck(innerE == e && JavaReferenceEqual(innerOld, a), "new observer")
							return b
						})
						optimizationCheck(JavaReferenceEqual(inner, a), "new observer result")
					} else {
						optimizationCheck(JavaReferenceEqual(old, b), "outer retry")
					}
					return next
				})
			})
			if calls != 2 || !optimizationResultIs(api, result, b, next) || !JavaReferenceEqual(VolatileLoad[any](&p.r), next) {
				t.Fatal("new observer hid invalidation")
			}
		})
	}
}

func TestAtomicOptimizationBehavior07Nonmutations(t *testing.T) {
	for _, api := range optimizationAPIs {
		t.Run(api.name, func(t *testing.T) {
			p, e, u := optimizationFixture()
			a, b, next, operand := NewObject(), NewObject(), NewObject(), NewObject()
			VolatileStore(&p.r, a)
			calls := 0
			result := optimizationComplete(t, func() any {
				return api.invoke(u, e, p, operand, func(actual *Execution, old any) any {
					calls++
					optimizationCheck(calls == 1 && actual == e && JavaReferenceEqual(old, a), "nonmutation retry")
					optimizationCheck(!u.CompareAndSetExecution(e, p, b, next), "failed CAS wrote")
					optimizationCheck(JavaReferenceEqual(u.GetExecution(e, p), a), "plain read")
					VolatileStore(&p.other, int32(7))
					return next
				})
			})
			if calls != 1 || !optimizationResultIs(api, result, a, next) || !JavaReferenceEqual(VolatileLoad[any](&p.r), next) || VolatileLoad[int32](&p.other) != 7 {
				t.Fatal("nonmutation invalidated cell")
			}
		})
	}
}

type optimizationProvider struct {
	info   *ObjectInfo
	onInfo func()
	calls  int
}

func (*optimizationProvider) JavaDynamicTypeID() TypeID { return ObjectTypeID }
func (p *optimizationProvider) JavaObjectInfo() *ObjectInfo {
	p.calls++
	if p.onInfo != nil {
		p.onInfo()
	}
	return p.info
}

func TestAtomicOptimizationBehavior08ProviderReentrantWrite(t *testing.T) {
	for _, api := range optimizationAPIs {
		for _, stage := range []string{"old_view", "new_value"} {
			t.Run(api.name+"/"+stage, func(t *testing.T) {
				p, e, u := optimizationFixture()
				a, b, operand := NewObject(), NewObject(), NewObject()
				calls := 0
				fired := false
				probe := &optimizationProvider{}
				probe.onInfo = func() {
					if !fired {
						fired = true
						VolatileStore(&p.r, b)
					}
				}
				var old, next any = a, NewObject()
				if stage == "old_view" {
					old = probe
				} else {
					next = probe
				}
				VolatileStore(&p.r, old)
				result := optimizationComplete(t, func() any {
					return api.invoke(u, e, p, operand, func(actual *Execution, value any) any {
						calls++
						optimizationCheck(calls <= 2 && actual == e, "provider callback")
						if calls == 1 {
							optimizationCheck(value == old, "provider first view")
						} else {
							optimizationCheck(value == b, "provider latest")
						}
						return next
					})
				})
				wantProviders := 1
				if stage == "new_value" {
					wantProviders = 4
				}
				if !fired || calls != 2 || probe.calls != wantProviders || !optimizationResultIs(api, result, b, next) || VolatileLoad[any](&p.r) != next {
					t.Fatal("provider reentrancy/result/trace changed")
				}
			})
		}
	}
}

func TestAtomicOptimizationBehavior09AbruptAfterWrite(t *testing.T) {
	for _, api := range optimizationAPIs {
		for _, stage := range []string{"callback", "wrong_type", "old_provider", "new_provider"} {
			t.Run(api.name+"/"+stage, func(t *testing.T) {
				p, e, u := optimizationFixture()
				a, b, operand := NewObject(), NewObject(), NewObject()
				marker := NewIllegalStateException("optimization abrupt marker")
				calls := 0
				probe := &optimizationProvider{}
				probe.onInfo = func() { VolatileStore(&p.r, b); panic(marker) }
				var old any = a
				if stage == "old_provider" {
					old = probe
				}
				VolatileStore(&p.r, old)
				failure := optimizationComplete(t, func() any {
					return optimizationFailure(func() {
						api.invoke(u, e, p, operand, func(actual *Execution, _ any) any {
							calls++
							optimizationCheck(actual == e && calls == 1, "abrupt callback")
							switch stage {
							case "wrong_type":
								VolatileStore(&p.r, b)
								return int32(1)
							case "new_provider":
								return probe
							default:
								VolatileStore(&p.r, b)
								panic(marker)
							}
						})
					})
				})
				if stage == "wrong_type" {
					id, ok := ObjectDynamicType(failure)
					if !ok || id != BuiltinThrowableTypeID("ClassCastException") {
						t.Fatal("wrong result failure")
					}
				} else if failure != marker {
					t.Fatal("abrupt identity")
				}
				wantCalls := 1
				if stage == "old_provider" {
					wantCalls = 0
				}
				if calls != wantCalls || VolatileLoad[any](&p.r) != b {
					t.Fatal("failed operation overwrote competing side effect")
				}
			})
		}
	}
}

func TestAtomicOptimizationBehavior10PrimitiveCASAndAccess(t *testing.T) {
	optimizationComplete(t, func() any {
		p, e, _ := optimizationFixture()
		i := NewAtomicIntegerFieldUpdater(ClassLiteral(optimizationOwner), "i", optimizationOwner)
		l := NewAtomicLongFieldUpdater(ClassLiteral(optimizationOwner), "l", optimizationOwner)
		optimizationCheck(i.CompareAndSetExecution(e, p, 0, 0) && l.CompareAndSetExecution(e, p, 0, 0), "default CAS")
		for _, v := range []int32{0, 1, -1, 2147483647, -2147483648} {
			i.SetExecution(e, p, v)
			optimizationCheck(i.CompareAndSetExecution(e, p, v, v) && !i.CompareAndSetExecution(e, p, v+1, 8) && i.GetExecution(e, p) == v, "int CAS")
		}
		for _, v := range []int64{0, 1, -1, 9223372036854775807, -9223372036854775808} {
			l.SetExecution(e, p, v)
			optimizationCheck(l.WeakCompareAndSetExecution(e, p, v, v) && !l.CompareAndSetExecution(e, p, v+1, 8) && l.GetExecution(e, p) == v, "long CAS")
		}
		VolatileStore(&p.i, "wrong primitive representation")
		optimizationCheck(!i.CompareAndSetExecution(e, p, 0, 1) && VolatileLoad[string](&p.i) == "wrong primitive representation", "wrong storage CAS")
		VolatileStore(&p.l, int32(0))
		optimizationCheck(!l.CompareAndSetExecution(e, p, 0, 1) && VolatileLoad[int32](&p.l) == 0, "wrong width CAS")
		i.SetExecution(e, p, 2147483647)
		optimizationCheck(i.GetAndAddExecution(e, p, 1) == 2147483647 && i.GetExecution(e, p) == -2147483648, "int wrap")
		l.SetExecution(e, p, -9223372036854775808)
		optimizationCheck(l.GetAndAddExecution(e, p, -1) == -9223372036854775808 && l.GetExecution(e, p) == 9223372036854775807, "long wrap")
		for _, action := range []func(){func() { i.CompareAndSetExecution(e, nil, 0, 1) }, func() { l.CompareAndSetExecution(e, NewObject(), 0, 1) }} {
			failure := optimizationFailure(action)
			id, ok := ObjectDynamicType(failure)
			optimizationCheck(ok && id == BuiltinThrowableTypeID("ClassCastException"), "receiver check")
		}
		child := TypeID("optimization.foreign.Child")
		RegisterJavaType(child, optimizationOwner)
		RegisterClassDescriptor(ClassDescriptor{Type: child, HasModifiers: true, Modifiers: 1, Fields: []FieldDescriptor{{Name: "i", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(e *Execution, v any) *VolatileFieldCell { return &v.(*optimizationTarget).shadow }}}})
		p.info = NewObjectInfo(child, func(TypeID) any { return p })
		ci := NewAtomicIntegerFieldUpdater(ClassLiteral(child), "i", child)
		ci.SetExecution(e, p, 3)
		optimizationCheck(ci.CompareAndSetExecution(e, p, 3, 4) && i.GetExecution(e, p) == -2147483648, "declaring shadow")
		d := classDescriptor(optimizationOwner)
		d.Fields = append(d.Fields, FieldDescriptor{Name: "protected", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 68, VolatileCell: func(e *Execution, v any) *VolatileFieldCell { return &v.(*optimizationTarget).i }})
		RegisterClassDescriptor(d)
		protected := NewAtomicIntegerFieldUpdater(ClassLiteral(optimizationOwner), "protected", child)
		optimizationCheck(protected.CompareAndSetExecution(e, p, -2147483648, 2), "protected allowed")
		failure := optimizationFailure(func() { protected.CompareAndSetExecution(e, &optimizationTarget{}, 0, 1) })
		id, ok := ObjectDynamicType(failure)
		optimizationCheck(ok && id == BuiltinThrowableTypeID("RuntimeException"), "protected denied")
		cause, known := ObjectDynamicType(GetCause(failure))
		optimizationCheck(known && cause == BuiltinThrowableTypeID("IllegalAccessException"), "protected cause")
		return nil
	})
}

func TestAtomicOptimizationBehavior11ResolverTrace(t *testing.T) {
	optimizationComplete(t, func() any {
		p, e, u := optimizationFixture()
		i := NewAtomicIntegerFieldUpdater(ClassLiteral(optimizationOwner), "i", optimizationOwner)
		l := NewAtomicLongFieldUpdater(ClassLiteral(optimizationOwner), "l", optimizationOwner)
		calls := 0
		p.onResolve = func(actual *Execution, name string) {
			calls++
			optimizationCheck(actual == e, "trace Execution")
			if calls == 2 {
				if name == "i" {
					VolatileStore(&p.i, int32(7))
				} else if name == "l" {
					VolatileStore(&p.l, int64(7))
				}
			}
		}
		optimizationCheck(i.GetAndAddExecution(e, p, 1) == 7 && calls == 3 && VolatileLoad[int32](&p.i) == 8, "int add resolver trace")
		calls = 0
		optimizationCheck(l.GetAndAddExecution(e, p, 1) == 7 && calls == 3 && VolatileLoad[int64](&p.l) == 8, "long add resolver trace")
		calls = 0
		operand := NewObject()
		next := NewObject()
		callbackCalls := 0
		result := u.AccumulateAndGetExecution(e, p, operand, func(actual *Execution, old, x any) any {
			callbackCalls++
			optimizationCheck(actual == e && x == operand && nilJavaReference(old), "accumulator trace")
			return next
		})
		optimizationCheck(result == next && calls == 2 && callbackCalls == 1, "reference accumulator resolutions")
		return nil
	})
}

type optimizationNode struct{ index int }

func (*optimizationNode) JavaDynamicTypeID() TypeID { return ObjectTypeID }

func TestAtomicOptimizationBehavior12StatefulAliasLifetime(t *testing.T) {
	// Matching original Java/generated-program gate is a separate mandatory gate.
	// This native control retains shared primitive cells and per-worker rings.
	optimizationComplete(t, func() any {
		p, _, u := optimizationFixture()
		p.want = nil
		i := NewAtomicIntegerFieldUpdater(ClassLiteral(optimizationOwner), "i", optimizationOwner)
		ia := NewAtomicIntegerFieldUpdater(ClassLiteral(optimizationOwner), "i", optimizationOwner)
		l := NewAtomicLongFieldUpdater(ClassLiteral(optimizationOwner), "l", optimizationOwner)
		const workers, operations, pool = 4, 128, 17
		type result struct {
			sumI, sumL int64
			failure    any
		}
		done := make(chan result, workers)
		for worker := 0; worker < workers; worker++ {
			go func() {
				r := result{}
				defer func() { r.failure = recover(); done <- r }()
				e := NewExecution()
				ring := &optimizationTarget{want: e}
				nodes := make([]*optimizationNode, pool)
				for k := range nodes {
					nodes[k] = &optimizationNode{index: k}
				}
				VolatileStore(&ring.r, nodes[0])
				operand := NewObject()
				for k := 0; k < operations; k++ {
					updater := i
					if k&1 != 0 {
						updater = ia
					}
					r.sumI += int64(updater.GetAndAddExecution(e, p, 1))
					r.sumL += l.GetAndAddExecution(e, p, 1)
					api := optimizationAPIs[k%len(optimizationAPIs)]
					oldIndex := k % pool
					nextIndex := (k + 1) % pool
					callbacks := 0
					value := api.invoke(u, e, ring, operand, func(actual *Execution, old any) any {
						callbacks++
						optimizationCheck(actual == e && callbacks == 1 && old == nodes[oldIndex], "stateful ring callback")
						return nodes[nextIndex]
					})
					optimizationCheck(optimizationResultIs(api, value, nodes[oldIndex], nodes[nextIndex]), "stateful old/new return")
				}
				optimizationCheck(VolatileLoad[any](&ring.r) == nodes[operations%pool], "retained ring final")
			}()
		}
		var sumI, sumL int64
		for w := 0; w < workers; w++ {
			r := <-done
			if r.failure != nil {
				panic(r.failure)
			}
			sumI += r.sumI
			sumL += r.sumL
		}
		const n = workers * operations
		wantSum := int64(n * (n - 1) / 2)
		optimizationCheck(VolatileLoad[int32](&p.i) == n && VolatileLoad[int64](&p.l) == n && sumI == wantSum && sumL == wantSum, "shared lifetime ticket oracle")
		return nil
	})
}

func TestAtomicOptimizationAllocationObservation(t *testing.T) {
	cell := &VolatileFieldCell{}
	token := &optimizationNode{index: 7}
	allocations := testing.AllocsPerRun(1024, func() { VolatileStore(cell, token) })
	if VolatileLoad[*optimizationNode](cell) != token {
		t.Fatal("allocation observation lost stored identity")
	}
	t.Logf("native runtime diagnostic: repeated same-value write without RMW observer = %.6f allocations/write; no throughput or generated-program claim", allocations)
}

func TestAtomicOptimizationAllocationWriteOnlyRegression(t *testing.T) {
	cell := &VolatileFieldCell{}
	token := &optimizationNode{index: 7}
	allocations := testing.AllocsPerRun(1024, func() { VolatileStore(cell, token) })
	if VolatileLoad[*optimizationNode](cell) != token {
		t.Fatal("allocation regression lost stored identity")
	}
	if allocations != 0 {
		t.Fatalf("write-only native allocation regression: got %.6f allocations/write, want zero with identity retained", allocations)
	}
}
