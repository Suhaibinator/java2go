package stdjava

import "strings"

func reflectionCheckedAccessible(member JavaAccessibleObject) JavaAccessibleObject {
	ReferenceRequireNonNull(member)
	if !ObjectInstanceOf(member, reflectionAccessibleObjectTypeID) {
		panic(NewClassCastException("value is not an AccessibleObject"))
	}
	return member
}
func reflectionMemberDescription(member JavaAccessibleObject) string {
	modifiers := member.GetModifiers()
	// Modifier.toString receives the declaration-kind mask. Bridge/varargs bits
	// overlap volatile/transient but are not Java source method modifiers.
	switch member.(type) {
	case *Field:
		modifiers &= 0x00df
	case *Method:
		modifiers &= 0x0d3f
	case *Constructor:
		modifiers &= 7
	}
	words := []string{}
	for _, entry := range []struct {
		bit  int32
		name string
	}{{1, "public"}, {4, "protected"}, {2, "private"}, {1024, "abstract"}, {8, "static"}, {16, "final"}, {128, "transient"}, {64, "volatile"}, {32, "synchronized"}, {256, "native"}, {2048, "strictfp"}} {
		if modifiers&entry.bit != 0 {
			words = append(words, entry.name)
		}
	}
	if _, method := member.(*Method); method && classDescriptor(member.GetDeclaringClass().TypeID()).Interface && member.GetModifiers()&1 != 0 && member.GetModifiers()&(1024|8|2) == 0 {
		words = append(words, "default")
	}
	prefix := strings.Join(words, " ")
	if prefix != "" {
		prefix += " "
	}
	owner := member.GetDeclaringClass().GetName()
	switch value := member.(type) {
	case *Field:
		return prefix + value.GetType().GetTypeName() + " " + owner + "." + value.GetName()
	case *Method:
		arguments := make([]string, len(value.descriptor.ParameterTypes))
		for index, id := range value.descriptor.ParameterTypes {
			arguments[index] = ClassLiteral(id).GetTypeName()
		}
		return prefix + value.GetReturnType().GetTypeName() + " " + owner + "." + value.GetName() + "(" + strings.Join(arguments, ",") + ")"
	case *Constructor:
		arguments := make([]string, len(value.descriptor.Parameters))
		for index, id := range value.descriptor.Parameters {
			arguments[index] = ClassLiteral(id).GetTypeName()
		}
		return prefix + owner + "(" + strings.Join(arguments, ",") + ")"
	}
	panic(NewUnsupportedOperationException("AccessibleObject member description is unavailable"))
}
func reflectionValidateAccessReceiver(member JavaAccessibleObject, receiver any) {
	_, constructor := member.(*Constructor)
	if constructor || member.GetModifiers()&8 != 0 {
		if !nilJavaReference(receiver) {
			panic(NewIllegalArgumentException("non-null object for " + reflectionMemberDescription(member)))
		}
		return
	}
	if nilJavaReference(receiver) {
		panic(NewIllegalArgumentException("null object for " + reflectionMemberDescription(member)))
	}
	id, known := ObjectDynamicType(receiver)
	if !known || !JavaTypeAssignable(id, member.GetDeclaringClass().TypeID()) {
		panic(NewIllegalArgumentException("object is not an instance of " + member.GetDeclaringClass().GetName()))
	}
}
func AccessibleObjectCanAccessExecution(execution *Execution, caller TypeID, member JavaAccessibleObject, receiver any) bool {
	requireExecution(execution)
	member = reflectionCheckedAccessible(member)
	reflectionValidateAccessReceiver(member, receiver)
	if member.reflectionAccessState().Load() {
		return true
	}
	return reflectionCallerAccess(caller, member, receiver)
}
func reflectionCanSetAccessible(caller TypeID, member JavaAccessibleObject) bool {
	owner := member.GetDeclaringClass()
	location, module := reflectionClassModule(owner.TypeID())
	if _, constructor := member.(*Constructor); constructor && owner.TypeID() == TypeID("java.lang.Class") {
		panic(reflectionException("SecurityException", "Cannot make a java.lang.Class constructor accessible"))
	}
	if caller == "" {
		targets, present := module.Exports[reflectionPackage(owner.TypeID())]
		return owner.GetModifiers()&1 != 0 && member.GetModifiers()&1 != 0 && (module.Name == "" || module.Open || present && len(targets) == 0)
	}
	callerLocation, _ := reflectionClassModule(caller)
	if reflectionSameModule(callerLocation, location) || callerLocation.ModuleName == "java.base" && callerLocation.LoaderIdentity == BootstrapClassLoaderIdentity || module.Name == "" {
		return true
	}
	packageName := reflectionPackage(owner.TypeID())
	if owner.GetModifiers()&1 != 0 && reflectionModuleEdge(module, packageName, callerLocation, false) {
		if member.GetModifiers()&1 != 0 {
			return true
		}
		if member.GetModifiers()&(4|8) == (4|8) && JavaTypeAssignable(caller, owner.TypeID()) {
			return true
		}
	}
	return reflectionModuleEdge(module, packageName, callerLocation, true)
}
func reflectionAccessDenied(caller TypeID, member JavaAccessibleObject) {
	owner := member.GetDeclaringClass()
	location := ClassRuntimeLocationOf(owner.TypeID())
	kind := ""
	if _, field := member.(*Field); field {
		kind = "field "
	}
	message := "Unable to make " + kind + reflectionMemberDescription(member) + " accessible"
	if caller == "" {
		message += " by JNI attached native thread with no caller frame: "
	} else {
		message += ": "
	}
	operation := "opens"
	if owner.GetModifiers()&1 != 0 && member.GetModifiers()&1 != 0 {
		operation = "exports"
	}
	message += "module " + location.ModuleName + " does not \"" + operation + " " + reflectionPackage(owner.TypeID()) + "\""
	if caller != "" {
		callerLocation := ClassRuntimeLocationOf(caller)
		message += " to "
		if callerLocation.ModuleName != "" {
			message += "module " + callerLocation.ModuleName
		} else {
			message += reflectionUnnamedModuleText(callerLocation.LoaderIdentity)
		}
	}
	panic(reflectionException("InaccessibleObjectException", message))
}
func AccessibleObjectSetAccessibleExecution(execution *Execution, caller TypeID, member JavaAccessibleObject, enabled bool) {
	requireExecution(execution)
	member = reflectionCheckedAccessible(member)
	if enabled && !reflectionCanSetAccessible(caller, member) {
		reflectionAccessDenied(caller, member)
	}
	member.reflectionAccessState().Store(enabled)
}
func AccessibleObjectTrySetAccessibleExecution(execution *Execution, caller TypeID, member JavaAccessibleObject) bool {
	requireExecution(execution)
	member = reflectionCheckedAccessible(member)
	if member.reflectionAccessState().Load() {
		return true
	}
	if _, constructor := member.(*Constructor); constructor && member.GetDeclaringClass().TypeID() == TypeID("java.lang.Class") {
		return false
	}
	if !reflectionCanSetAccessible(caller, member) {
		return false
	}
	member.reflectionAccessState().Store(true)
	return true
}
func AccessibleObjectSetAccessibleArrayExecution(execution *Execution, caller TypeID, array *ReferenceArray, enabled bool) {
	requireExecution(execution)
	if array == nil {
		message := "Cannot read the array length because \"<local2>\" is null"
		if enabled {
			message = "Cannot invoke \"[Ljava.lang.reflect.AccessibleObject;.clone()\" because \"array\" is null"
		}
		panic(NewNullPointerException(message))
	}
	values := array.elements
	if enabled {
		values = append([]any(nil), values...)
		for _, value := range values {
			if nilJavaReference(value) {
				panic(NewNullPointerException("Cannot invoke \"java.lang.reflect.AccessibleObject.checkCanSetAccessible(java.lang.Class)\" because \"ao\" is null"))
			}
			member := ObjectView[JavaAccessibleObject](value, reflectionAccessibleObjectTypeID)
			if !reflectionCanSetAccessible(caller, member) {
				reflectionAccessDenied(caller, member)
			}
		}
	}
	for _, value := range values {
		if nilJavaReference(value) {
			panic(NewNullPointerException("Cannot invoke \"java.lang.reflect.AccessibleObject.setAccessible0(boolean)\" because \"ao\" is null"))
		}
		ObjectView[JavaAccessibleObject](value, reflectionAccessibleObjectTypeID).reflectionAccessState().Store(enabled)
	}
}
