package stdjava

import (
	"reflect"
	"testing"
)

const atomicNestOwner TypeID = "atomic.nest.Host$Holder"

type atomicNestTarget struct{ i, l, r VolatileFieldCell }

func (*atomicNestTarget) JavaDynamicTypeID() TypeID { return atomicNestOwner }

// Set the proposed additive descriptor field only when available. This keeps
// the unchanged baseline compilable so RED measures access semantics, rather
// than an unknown-field compiler error. No binary-name inference occurs here.
func atomicNestDescriptor(id, host TypeID) ClassDescriptor {
	descriptor := ClassDescriptor{Type: id, HasModifiers: true, Modifiers: 1}
	field := reflect.ValueOf(&descriptor).Elem().FieldByName("NestHost")
	if field.IsValid() {
		field.Set(reflect.ValueOf(host))
	}
	return descriptor
}
func TestAtomicUpdaterDeclaredNestAccessTDD(t *testing.T) {
	host := TypeID("atomic.nest.Host")
	for _, declaration := range []struct{ id, host TypeID }{
		{host, host}, {atomicNestOwner, host}, {"atomic.nest.Host$Sibling", host},
		{"atomic.nest.Unrelated", "atomic.nest.Unrelated"},
		{"atomic.nest.Host$Spoof", "atomic.nest.Host$Spoof"},
	} {
		RegisterJavaType(declaration.id, ObjectTypeID)
		descriptor := atomicNestDescriptor(declaration.id, declaration.host)
		if declaration.id == atomicNestOwner {
			descriptor.Fields = []FieldDescriptor{
				{Name: "i", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 66, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*atomicNestTarget).i }},
				{Name: "l", Type: PrimitiveLongTypeID, HasModifiers: true, Modifiers: 66, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*atomicNestTarget).l }},
				{Name: "r", Type: ObjectTypeID, HasModifiers: true, Modifiers: 66, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*atomicNestTarget).r }},
			}
		}
		RegisterClassDescriptor(descriptor)
	}
	for _, test := range []struct {
		name    string
		caller  TypeID
		allowed bool
	}{
		{"outer_positive", host, true}, {"sibling_positive", "atomic.nest.Host$Sibling", true},
		{"unrelated_negative", "atomic.nest.Unrelated", false},
		{"literal_dollar_top_level_negative", "atomic.nest.Host$Spoof", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, family := range []string{"integer", "long", "reference"} {
				t.Run(family, func(t *testing.T) {
					target := &atomicNestTarget{}
					exercise := func() {
						switch family {
						case "integer":
							u := NewAtomicIntegerFieldUpdater(ClassLiteral(atomicNestOwner), "i", test.caller)
							u.Set(target, 7)
							if u.Get(target) != 7 || VolatileLoad[int32](&target.i) != 7 {
								t.Fatal("declaring integer cell lost")
							}
						case "long":
							u := NewAtomicLongFieldUpdater(ClassLiteral(atomicNestOwner), "l", test.caller)
							u.Set(target, 9)
							if u.Get(target) != 9 || VolatileLoad[int64](&target.l) != 9 {
								t.Fatal("declaring long cell lost")
							}
						case "reference":
							u := NewAtomicReferenceFieldUpdater(ClassLiteral(atomicNestOwner), ClassLiteral(ObjectTypeID), "r", test.caller)
							value := BoxInteger(700)
							u.Set(target, value)
							if !JavaReferenceEqual(u.Get(target), value) || !JavaReferenceEqual(VolatileLoad[any](&target.r), value) {
								t.Fatal("declaring reference cell lost")
							}
						}
					}
					if test.allowed {
						var failure any
						func() { defer func() { failure = recover() }(); exercise() }()
						if failure != nil {
							t.Fatalf("declared nest access rejected: %v", failure)
						}
					} else {
						denied := updaterFailure(t, BuiltinThrowableTypeID("RuntimeException"), exercise)
						if cause, ok := ObjectDynamicType(GetCause(denied)); !ok || cause != BuiltinThrowableTypeID("IllegalAccessException") {
							t.Fatal("private denial cause")
						}
					}
				})
			}
		})
	}
}
