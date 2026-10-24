package stdjava

import (
	"reflect"
	"strings"
)

const ReflectMethodTypeID TypeID = "java.lang.reflect.Method"

func init() {
	RegisterJavaType(ReflectMethodTypeID, reflectionExecutableTypeID, reflectionMemberTypeID)
}
func (method *Method) JavaDynamicTypeID() TypeID { return ReflectMethodTypeID }
func (method *Method) requireMethod() {
	if method == nil {
		panic(NewNullPointerException("method is null"))
	}
}
func (method *Method) GetReturnType() *Class {
	method.requireMethod()
	id := method.descriptor.Return
	if id == "" || id == "void" {
		id = PrimitiveTypeID("void")
	}
	return ClassLiteral(id)
}
func (method *Method) GetDeclaringClass() *Class { method.requireMethod(); return method.owner }
func (method *Method) IsBridge() bool            { method.requireMethod(); return method.descriptor.Bridge }
func (method *Method) IsSynthetic() bool         { method.requireMethod(); return method.descriptor.Synthetic }
func (method *Method) GetModifiers() int32 {
	method.requireMethod()
	if !method.descriptor.HasModifiers {
		return 1
	}
	return method.descriptor.Modifiers
}
func (method *Method) SetAccessible(value bool) {
	method.requireMethod()
	method.accessible.Store(value)
}
func (method *Method) CanAccess(receiver any) bool {
	method.requireMethod()
	modifiers := method.GetModifiers()
	// AccessibleObject validates its receiver domain even when access checks
	// have been overridden. Method.invoke has a different static receiver rule.
	if modifiers&8 != 0 {
		if !nilJavaReference(receiver) {
			panic(NewIllegalArgumentException("non-null receiver for static method"))
		}
	} else {
		if nilJavaReference(receiver) {
			panic(NewIllegalArgumentException("null receiver for instance method"))
		}
		reflectionReceiver(receiver, method.owner)
	}
	return method.accessible.Load() || modifiers&1 != 0
}
func reflectionMethodParametersMatch(descriptor MethodDescriptor, parameters []*Class) bool {
	if len(descriptor.ParameterTypes) != len(parameters) {
		return false
	}
	for index, parameter := range parameters {
		if parameter == nil || parameter.TypeID() != descriptor.ParameterTypes[index] {
			return false
		}
	}
	return true
}
func (class *Class) reflectionFindMethod(name string, parameters []*Class, declared bool) *Method {
	return class.reflectionFindMethodMatching(func(d MethodDescriptor) bool { return d.Name == name }, parameters, declared)
}
func (class *Class) reflectionFindMethodJavaStringKey(key string, parameters []*Class, declared bool) *Method {
	return class.reflectionFindMethodMatching(func(d MethodDescriptor) bool { return JavaStringSwitchKey(d.nameJavaString) == key }, parameters, declared)
}
func (class *Class) reflectionFindMethodMatching(nameMatches func(MethodDescriptor) bool, parameters []*Class, declared bool) *Method {
	queue := []TypeID{class.TypeID()}
	seen := make(map[TypeID]bool)
	var inheritedInterface *Method
	for len(queue) != 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		var best *Method
		for _, descriptor := range classDescriptor(id).Methods {
			modifiers := descriptor.Modifiers
			if !descriptor.HasModifiers {
				modifiers = 1
			}
			// Static interface declarations belong only to the declaring interface.
			inheritedInterfaceStatic := id != class.TypeID() && classDescriptor(id).Interface && modifiers&8 != 0
			if !nameMatches(descriptor) || !reflectionMethodParametersMatch(descriptor, parameters) || (!declared && (modifiers&1 == 0 || inheritedInterfaceStatic)) {
				continue
			}
			if best == nil || (JavaTypeAssignable(descriptor.Return, best.descriptor.Return) && descriptor.Return != best.descriptor.Return) || (descriptor.Return == best.descriptor.Return && best.descriptor.Bridge && !descriptor.Bridge) {
				best = &Method{owner: ClassLiteral(id), descriptor: descriptor}
			}
		}
		if best != nil {
			if declared || !classDescriptor(id).Interface || id == class.TypeID() {
				return best
			}
			if inheritedInterface == nil || reflectionMethodMoreSpecific(best, inheritedInterface) {
				inheritedInterface = best
			}
		}
		if declared {
			return nil
		}
		javaTypeRegistry.RLock()
		info := javaTypeRegistry.types[id]
		javaTypeRegistry.RUnlock()
		if info.super != "" {
			queue = append(queue, info.super)
		}
		queue = append(queue, info.interfaces...)
	}
	return inheritedInterface
}

// reflectionVirtualMethod follows overriding declarations from the base to the
// receiver class. Public reflection lookup has a different admission policy:
// protected and package methods can override, while private/static shadows cannot.
func (method *Method) reflectionVirtualMethod(dynamic TypeID) *Method {
	if method.GetModifiers()&(2|8|16) != 0 {
		return method
	}
	parameters := make([]*Class, len(method.descriptor.ParameterTypes))
	for index, id := range method.descriptor.ParameterTypes {
		parameters[index] = ClassLiteral(id)
	}
	matches := func(candidate MethodDescriptor) bool {
		return candidate.Name == method.descriptor.Name && candidate.Return == method.descriptor.Return
	}
	packageName := func(id TypeID) string {
		name := string(id)
		if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
			return name[:dot]
		}
		return ""
	}
	var hierarchy []TypeID
	seen := make(map[TypeID]bool)
	for id := dynamic; id != "" && !seen[id]; {
		seen[id] = true
		hierarchy = append(hierarchy, id)
		javaTypeRegistry.RLock()
		super := javaTypeRegistry.types[id].super
		javaTypeRegistry.RUnlock()
		id = super
	}
	selected := method
	interfaceMethod := classDescriptor(method.owner.TypeID()).Interface
	for index := len(hierarchy) - 1; index >= 0; index-- {
		id := hierarchy[index]
		if classDescriptor(id).Interface || (!interfaceMethod && !JavaTypeAssignable(id, method.owner.TypeID())) {
			continue
		}
		for _, candidate := range classDescriptor(id).Methods {
			modifiers := candidate.Modifiers
			if !candidate.HasModifiers {
				modifiers = 1
			}
			if modifiers&(2|8) != 0 || !matches(candidate) || !reflectionMethodParametersMatch(candidate, parameters) {
				continue
			}
			if selected.GetModifiers()&16 != 0 {
				return selected
			}
			// A package declaration is overridden only in its declaring package,
			// or through a later public/protected override in that same package.
			if selected.GetModifiers()&5 == 0 && packageName(id) != packageName(selected.owner.TypeID()) {
				continue
			}
			if classDescriptor(selected.owner.TypeID()).Interface && modifiers&1 == 0 {
				continue
			}
			selected = &Method{owner: ClassLiteral(id), descriptor: candidate}
		}
	}
	if classDescriptor(selected.owner.TypeID()).Interface {
		if inherited := ClassLiteral(dynamic).reflectionFindMethodMatching(matches, parameters, false); inherited != nil {
			return inherited
		}
	}
	return selected
}
func reflectionMethodMoreSpecific(candidate, current *Method) bool {
	if candidate.descriptor.Return != current.descriptor.Return {
		return JavaTypeAssignable(candidate.descriptor.Return, current.descriptor.Return)
	}
	return JavaTypeAssignable(candidate.owner.TypeID(), current.owner.TypeID()) || current.descriptor.Bridge && !candidate.descriptor.Bridge
}
func (class *Class) GetDeclaredMethod(name string, parameters ...*Class) *Method {
	class.TypeID()
	if nilJavaReference(name) {
		panic(NewNullPointerException("method name is null"))
	}
	if method := class.reflectionFindMethod(name, parameters, true); method != nil {
		return method
	}
	panic(reflectionException("NoSuchMethodException", name))
}
func (class *Class) GetDeclaredMethodJavaString(name *JavaString, parameters ...*Class) *Method {
	class.TypeID()
	if name == nil {
		panic(NewNullPointerException("method name is null"))
	}
	if method := class.reflectionFindMethodJavaStringKey(JavaStringSwitchKey(name), parameters, true); method != nil {
		return method
	}
	panic(reflectionJavaStringException("NoSuchMethodException", name))
}
func (class *Class) GetDeclaredMethods() *ReferenceArray {
	descriptors := classDescriptor(class.TypeID()).Methods
	array := NewReferenceArray(len(descriptors), ReflectMethodTypeID)
	for index, descriptor := range descriptors {
		ReferenceArraySet(array, index, &Method{owner: class, descriptor: descriptor})
	}
	return array
}

func (method *Method) Invoke(execution *Execution, receiver any, arguments ...any) any {
	method.requireMethod()
	requireExecution(execution)
	static := method.GetModifiers()&8 != 0
	if !static && nilJavaReference(receiver) {
		panic(NewNullPointerException("reflection receiver is null"))
	}
	if !method.accessible.Load() && method.GetModifiers()&1 == 0 {
		panic(reflectionException("IllegalAccessException", method.descriptor.Name))
	}
	if len(arguments) != len(method.descriptor.ParameterTypes) {
		panic(NewIllegalArgumentException("wrong method argument count"))
	}
	converted := make([]any, len(arguments))
	for index, argument := range arguments {
		converted[index] = reflectionFieldValue(argument, method.descriptor.ParameterTypes[index])
	}
	descriptor := method.descriptor
	owner := method.owner
	var target reflect.Value
	if static {
		if descriptor.StaticFunction == nil {
			panic(NewUnsupportedOperationException("static method body is unavailable"))
		}
		target = reflect.ValueOf(descriptor.StaticFunction)
	} else {
		// Method.invoke dispatches virtually even when the Method came from a superclass.
		reflectionReceiver(receiver, owner)
		if dynamic, ok := ObjectDynamicType(receiver); ok {
			// Selection retains the exact return descriptor, including erased bridges.
			selected := method.reflectionVirtualMethod(dynamic)
			descriptor, owner = selected.descriptor, selected.owner
		}
		target = reflectionReceiver(receiver, owner).MethodByName(descriptor.GoName)
	}
	if !target.IsValid() || target.Kind() != reflect.Func || target.Type().NumIn() != len(arguments)+1 || target.Type().In(0) != reflect.TypeOf((*Execution)(nil)) || target.Type().NumOut() > 1 {
		panic(NewUnsupportedOperationException("method execution signature is unavailable"))
	}
	if static {
		if initialize := classDescriptor(owner.TypeID()).Initialize; initialize != nil {
			initialize(execution)
		}
	}
	defer reflectionInvocationPanic()
	values := []reflect.Value{reflect.ValueOf(execution)}
	for index, value := range converted {
		if descriptor.Bridge && len(descriptor.InvocationParameterTypes) == len(converted) {
			id := descriptor.InvocationParameterTypes[index]
			if !nilJavaReference(value) {
				if actual, known := ObjectDynamicType(value); known && !JavaTypeAssignable(actual, id) {
					panic(NewClassCastException("bridge argument has wrong Java type"))
				}
			}
			value = reflectionFieldValue(value, id)
		}
		expected := target.Type().In(index + 1)
		argument := reflect.Zero(expected)
		if !nilJavaReference(value) {
			argument = reflect.ValueOf(value)
			if !argument.Type().AssignableTo(expected) {
				panic(NewUnsupportedOperationException("method argument representation is unavailable"))
			}
		}
		values = append(values, argument)
	}
	results := target.Call(values)
	if len(results) == 0 {
		return nil
	}
	result := results[0].Interface()
	if descriptor.Bridge && !nilJavaReference(result) && !isPrimitiveTypeID(descriptor.Return) {
		if actual, known := ObjectDynamicType(result); known && !JavaTypeAssignable(actual, descriptor.Return) {
			panic(NewClassCastException("bridge result has wrong Java type"))
		}
	}
	return reflectionBox(result, method.descriptor.Return)
}
