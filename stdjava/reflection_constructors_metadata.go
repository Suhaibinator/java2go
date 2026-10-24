package stdjava

import (
	"fmt"
	"strings"
)

func reflectionConstructorParameters(parameters []*Class) []TypeID {
	result := make([]TypeID, len(parameters))
	for index, parameter := range parameters {
		result[index] = parameter.TypeID()
	}
	return result
}
func reflectionConstructorCandidates(class *Class) []ConstructorDescriptor {
	descriptor := classDescriptor(class.TypeID())
	if descriptor.Constructors != nil {
		return descriptor.Constructors
	}
	if descriptor.Construct != nil {
		return []ConstructorDescriptor{{Modifiers: 1, Construct: func(execution *Execution, _ []any) any { return descriptor.Construct(execution) }}}
	}
	return nil
}
func reflectionParameterTypesEqual(left, right []TypeID) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
func reflectionConstructorLookup(class *Class, publicOnly bool, parameters []*Class) *Constructor {
	class.TypeID()
	ids := reflectionConstructorParameters(parameters)
	for _, descriptor := range reflectionConstructorCandidates(class) {
		if (!publicOnly || descriptor.Modifiers&1 != 0) && reflectionParameterTypesEqual(descriptor.Parameters, ids) {
			return &Constructor{owner: class, descriptor: descriptor}
		}
	}
	names := make([]string, len(ids))
	for index, id := range ids {
		names[index] = ClassLiteral(id).GetTypeName()
	}
	panic(reflectionException("NoSuchMethodException", class.GetName()+".<init>("+strings.Join(names, ",")+")"))
}
func (class *Class) GetConstructor(parameters ...*Class) *Constructor {
	return reflectionConstructorLookup(class, true, parameters)
}
func (class *Class) GetDeclaredConstructor(parameters ...*Class) *Constructor {
	return reflectionConstructorLookup(class, false, parameters)
}
func (constructor *Constructor) GetModifiers() int32 {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	return constructor.descriptor.Modifiers
}
func (constructor *Constructor) GetDeclaringClass() *Class {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	return constructor.owner
}
func (constructor *Constructor) SetAccessible(enabled bool) {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	constructor.accessible.Store(enabled)
}
func (constructor *Constructor) publicAccess() bool {
	return constructor.accessible.Load() || (constructor.GetModifiers()&1 != 0 && constructor.owner.GetModifiers()&1 != 0)
}
func (constructor *Constructor) CanAccess(receiver any) bool {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	if !nilJavaReference(receiver) {
		prefix := ""
		switch constructor.GetModifiers() & 7 {
		case 1:
			prefix = "public "
		case 2:
			prefix = "private "
		case 4:
			prefix = "protected "
		}
		names := make([]string, len(constructor.descriptor.Parameters))
		for index, id := range constructor.descriptor.Parameters {
			names[index] = ClassLiteral(id).GetTypeName()
		}
		panic(NewIllegalArgumentException("non-null object for " + prefix + constructor.owner.GetName() + "(" + strings.Join(names, ",") + ")"))
	}
	return constructor.publicAccess()
}
func (constructor *Constructor) NewInstance(execution *Execution, arguments ...any) any {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	requireExecution(execution)
	if !constructor.publicAccess() {
		panic(reflectionException("IllegalAccessException", "non-public constructor"))
	}
	descriptor := classDescriptor(constructor.owner.TypeID())
	if constructor.owner.GetModifiers()&1024 != 0 {
		panic(Exception{newJavaThrowableBase("InstantiationException", nil)})
	}
	if descriptor.Enum {
		panic(NewIllegalArgumentException("Cannot reflectively create enum objects"))
	}
	// Installed JDK21 accessor initialization precedes arity/unboxing rejection;
	// declaring-class initialization failures escape without a target wrapper.
	if descriptor.Initialize != nil {
		descriptor.Initialize(execution)
	}
	if len(arguments) != len(constructor.descriptor.Parameters) {
		panic(NewIllegalArgumentException(fmt.Sprintf("wrong number of arguments: %d expected: %d", len(arguments), len(constructor.descriptor.Parameters))))
	}
	converted := make([]any, len(arguments))
	for index, argument := range arguments {
		converted[index] = reflectionFieldValue(argument, constructor.descriptor.Parameters[index])
	}
	if constructor.descriptor.Construct == nil {
		panic(NewUnsupportedOperationException("constructor callback metadata is unavailable"))
	}
	defer reflectionInvocationPanic()
	return constructor.descriptor.Construct(execution, converted)
}
