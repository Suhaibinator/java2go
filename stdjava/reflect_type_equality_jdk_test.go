package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func reflectEqualityJDKObservation(t *testing.T, key, got string) {
	t.Helper()
	reflectEqualityJDKFileObservation(t, "RuntimeOracle", key, got)
}

func reflectEqualityJDKFileObservation(t *testing.T, name, key, got string) {
	t.Helper()
	data, err := os.ReadFile("testdata/reflect_type_equals131/" + name + ".expected.stdout")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.HasPrefix(line, key+"=") {
			if got != line {
				t.Fatalf("actual JDK %q != Go %q", line, got)
			}
			return
		}
	}
	t.Fatal("missing actual JDK observation", key)
}

type reflectEqualityEffect struct{ events *string }

func (*reflectEqualityEffect) JavaDynamicTypeID() TypeID { return "eqruntime.Effect" }
func (v *reflectEqualityEffect) EqualsJava2goExecution(_ *Execution, other any) bool {
	if javaReferenceIsNull(other) {
		*v.events += "N"
		return true
	}
	*v.events += "E"
	return false
}
func (*reflectEqualityEffect) HashCode() int32 { return 23 }

type reflectEqualityParameter struct {
	events     *string
	owner, raw ReflectType
	args       *ReferenceArray
	fail       bool
}

func (*reflectEqualityParameter) JavaDynamicTypeID() TypeID { return "eqruntime.Parameter" }
func (v *reflectEqualityParameter) GetOwnerTypeJava2goExecution(*Execution) ReflectType {
	*v.events += "O"
	return v.owner
}
func (v *reflectEqualityParameter) GetRawTypeJava2goExecution(*Execution) ReflectType {
	*v.events += "R"
	if v.fail {
		panic(NewIllegalStateException("raw"))
	}
	return v.raw
}
func (v *reflectEqualityParameter) GetActualTypeArgumentsJava2goExecution(*Execution) *ReferenceArray {
	*v.events += "A"
	return v.args
}

type reflectEqualityArray struct {
	events    *string
	component ReflectType
}

func (*reflectEqualityArray) JavaDynamicTypeID() TypeID { return "eqruntime.Array" }
func (v *reflectEqualityArray) GetGenericComponentTypeJava2goExecution(*Execution) ReflectType {
	*v.events += "G"
	return v.component
}

type reflectEqualityWild struct {
	events       *string
	lower, upper *ReferenceArray
}

func (*reflectEqualityWild) JavaDynamicTypeID() TypeID { return "eqruntime.Wild" }
func (v *reflectEqualityWild) GetLowerBoundsJava2goExecution(*Execution) *ReferenceArray {
	*v.events += "L"
	return v.lower
}
func (v *reflectEqualityWild) GetUpperBoundsJava2goExecution(*Execution) *ReferenceArray {
	*v.events += "U"
	return v.upper
}

type reflectEqualityForeign struct{ events *string }

func (*reflectEqualityForeign) JavaDynamicTypeID() TypeID { return "eqruntime.Foreign" }
func (v *reflectEqualityForeign) GetNameJava2goExecution(*Execution) *JavaString {
	*v.events += "N"
	return JavaStringFromHostUTF8("T")
}
func (v *reflectEqualityForeign) GetGenericDeclarationJava2goExecution(*Execution) GenericDeclaration {
	*v.events += "D"
	return ClassLiteral("eqruntime.Model")
}

func init() {
	RegisterJavaType("eqruntime.Effect", ObjectTypeID, ReflectTypeTypeID)
	RegisterJavaType("eqruntime.Parameter", ObjectTypeID, ParameterizedTypeTypeID)
	RegisterJavaType("eqruntime.Array", ObjectTypeID, GenericArrayTypeTypeID)
	RegisterJavaType("eqruntime.Wild", ObjectTypeID, WildcardTypeTypeID)
	RegisterJavaType("eqruntime.Foreign", ObjectTypeID, TypeVariableTypeID)
}

func TestReflectTypeEqualityComponentJDK21(t *testing.T) {
	e := NewExecution()
	events := ""
	effect := &reflectEqualityEffect{&events}
	left := &metadataGenericArrayType{component: effect}
	same := ObjectEqualsExecution(e, left, &reflectEqualityArray{&events, effect})
	sameEvents := events
	events = ""
	nilEqual := ObjectEqualsExecution(e, left, &reflectEqualityArray{&events, nil})
	reflectEqualityJDKObservation(t, "component", fmt.Sprintf("component=%t:%s:%t:%s", same, sameEvents, nilEqual, events))
}
func TestReflectTypeEqualityParameterizedOrderJDK21(t *testing.T) {
	e := NewExecution()
	events := ""
	left := &metadataParameterizedType{raw: ClassLiteral("java.util.List"), owner: ClassLiteral("eqruntime.Model"), arguments: []ReflectType{ClassLiteral(StringTypeID)}}
	result := "wrong"
	func() {
		defer func() {
			if caught := recover(); caught != nil {
				if exception, ok := caught.(IllegalStateException); ok {
					result = exception.Message()
				} else {
					panic(caught)
				}
			}
		}()
		ObjectEqualsExecution(e, left, &reflectEqualityParameter{events: &events, raw: ClassLiteral("java.util.List"), fail: true})
	}()
	reflectEqualityJDKObservation(t, "parameter", fmt.Sprintf("parameter=%s:%s", result, events))
}
func TestReflectTypeEqualityWildcardOrderJDK21(t *testing.T) {
	e := NewExecution()
	events := ""
	left := &metadataWildcardType{upper: []ReflectType{ClassLiteral(StringTypeID)}}
	other := &reflectEqualityWild{events: &events, lower: reflectTypeArray(nil), upper: reflectTypeArray([]ReflectType{ClassLiteral(StringTypeID)})}
	equal := ObjectEqualsExecution(e, left, other)
	reflectEqualityJDKObservation(t, "wild", fmt.Sprintf("wild=%t:%s", equal, events))
}
func TestReflectTypeEqualityNullArrayJDK21(t *testing.T) {
	e := NewExecution()
	events := ""
	left := &metadataParameterizedType{raw: ClassLiteral("java.util.List"), arguments: []ReflectType{ClassLiteral(StringTypeID)}}
	other := &reflectEqualityParameter{events: &events, raw: ClassLiteral("java.util.List")}
	equal := ObjectEqualsExecution(e, left, other)
	reflectEqualityJDKObservation(t, "null-array", fmt.Sprintf("null-array=%t:%s", equal, events))
}
func TestReflectTypeEqualityVariableFamilyJDK21(t *testing.T) {
	e := NewExecution()
	events := ""
	variable := &metadataTypeVariable{declaration: "eqruntime.Model", name: "T"}
	same := &metadataTypeVariable{declaration: "eqruntime.Model", name: "T"}
	peer := &metadataTypeVariable{declaration: "eqruntime.Peer", name: "T"}
	foreign := ObjectEqualsExecution(e, variable, &reflectEqualityForeign{&events})
	reflectEqualityJDKObservation(t, "variable", fmt.Sprintf("variable=%t:%s:%t:%t:%t", foreign, events, ObjectEqualsExecution(e, variable, same), !ObjectEqualsExecution(e, variable, peer), variable.HashCode() == same.HashCode()))
}
func TestReflectTypeEqualityIdentityJDK21(t *testing.T) {
	e := NewExecution()
	events := ""
	effect := &reflectEqualityEffect{&events}
	left := &metadataParameterizedType{raw: ClassLiteral("java.util.List"), owner: effect, arguments: []ReflectType{effect}}
	equal := ObjectEqualsExecution(e, left, left)
	reflectEqualityJDKObservation(t, "identity", fmt.Sprintf("identity=%t:%s", equal, events))
}

type reflectEqualityNullThrow struct {
	events    *string
	execution *Execution
}

func (*reflectEqualityNullThrow) JavaDynamicTypeID() TypeID { return "eqruntimeedges.Effect" }
func (value *reflectEqualityNullThrow) EqualsJava2goExecution(execution *Execution, other any) bool {
	if execution != value.execution {
		panic("lost caller Execution")
	}
	*value.events += "E"
	if javaReferenceIsNull(other) {
		panic(NewIllegalStateException("null"))
	}
	return false
}
func init() { RegisterJavaType("eqruntimeedges.Effect", ObjectTypeID, ReflectTypeTypeID) }
func TestReflectTypeEqualityNullThrowJDK21(t *testing.T) {
	execution := NewExecution()
	events := ""
	result := "wrong"
	left := &metadataGenericArrayType{component: &reflectEqualityNullThrow{&events, execution}}
	func() {
		defer func() {
			if caught := recover(); caught != nil {
				if exception, ok := caught.(IllegalStateException); ok {
					result = exception.Message()
				} else {
					panic(caught)
				}
			}
		}()
		ObjectEqualsExecution(execution, left, &reflectEqualityArray{&events, nil})
	}()
	reflectEqualityJDKFileObservation(t, "RuntimeEdgesOracle", "null-throw", fmt.Sprintf("null-throw=%s:%s", events, result))
}
func TestReflectTypeEqualityParameterizedOwnerJDK21(t *testing.T) {
	execution := NewExecution()
	owner := func(argument TypeID) ReflectType {
		return &metadataParameterizedType{raw: ClassLiteral("eqruntimeedges.RuntimeEdgesOracle$Outer"), arguments: []ReflectType{ClassLiteral(argument)}}
	}
	child := func(owner ReflectType) *metadataParameterizedType {
		return &metadataParameterizedType{raw: ClassLiteral("eqruntimeedges.RuntimeEdgesOracle$Outer$Inner"), arguments: []ReflectType{ClassLiteral("java.lang.Integer")}, owner: owner}
	}
	left, equal, unequal := child(owner(StringTypeID)), child(owner(StringTypeID)), child(owner("java.lang.Integer"))
	reflectEqualityJDKFileObservation(t, "RuntimeEdgesOracle", "owner", fmt.Sprintf("owner=%t:%t:%t", ObjectEqualsExecution(execution, left, equal), !ObjectEqualsExecution(execution, left, unequal), left.HashCode() == equal.HashCode()))
}
