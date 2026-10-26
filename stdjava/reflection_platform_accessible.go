package stdjava

// Platform descriptors below contain actual supported JDK declarations, never
// a substitute Gson helper. A partial platform inventory refuses unsupported
// enumeration/unknown lookup rather than pretending that the JDK member is absent.
func reflectionPlatformPartial(id TypeID) bool {
	return id == reflectionAccessibleObjectTypeID || id == AnnotationTypeID || id == StringTypeID || id == RecordTypeID || classDescriptor(id).PartialMemberMetadata
}
func reflectionPlatformMethodLookup(class *Class, matches func(string) bool) {
	if !reflectionPlatformPartial(class.TypeID()) {
		return
	}
	for _, method := range classDescriptor(class.TypeID()).Methods {
		if matches(method.Name) {
			return
		}
	}
	panic(NewUnsupportedOperationException("platform method metadata is outside the supported reflection surface"))
}
func reflectionPlatformFieldMissing(class *Class) {
	if reflectionPlatformPartial(class.TypeID()) {
		panic(NewUnsupportedOperationException("platform field metadata is outside the supported reflection surface"))
	}
}
func reflectionPlatformEnumeration(class *Class) {
	if reflectionPlatformPartial(class.TypeID()) {
		panic(NewUnsupportedOperationException("complete platform member metadata is unavailable"))
	}
}

func init() {
	location := ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}
	for _, id := range []TypeID{reflectionAccessibleObjectTypeID, reflectionExecutableTypeID, reflectionMemberTypeID, ReflectMethodTypeID, ReflectFieldTypeID, ReflectConstructorTypeID, AnnotationTypeID} {
		RegisterClassRuntimeLocation(id, location)
	}
	RegisterClassDescriptor(ClassDescriptor{Type: reflectionAccessibleObjectTypeID, RuntimeLocation: location, Modifiers: 1, HasModifiers: true, Methods: []MethodDescriptor{
		{Name: "canAccess", Return: PrimitiveBooleanTypeID, ParameterTypes: []TypeID{ObjectTypeID}, Modifiers: 17, HasModifiers: true, Invoke: func(execution *Execution, caller TypeID, receiver any, arguments []any) any {
			return AccessibleObjectCanAccessExecution(execution, caller, ObjectView[JavaAccessibleObject](receiver, reflectionAccessibleObjectTypeID), arguments[0])
		}},
		{Name: "setAccessible", Return: PrimitiveTypeID("void"), ParameterTypes: []TypeID{PrimitiveBooleanTypeID}, Modifiers: 1, HasModifiers: true, Invoke: func(execution *Execution, caller TypeID, receiver any, arguments []any) any {
			AccessibleObjectSetAccessibleExecution(execution, caller, ObjectView[JavaAccessibleObject](receiver, reflectionAccessibleObjectTypeID), arguments[0].(bool))
			return nil
		}},
		{Name: "setAccessible", Return: PrimitiveTypeID("void"), ParameterTypes: []TypeID{ArrayTypeID(reflectionAccessibleObjectTypeID), PrimitiveBooleanTypeID}, Modifiers: 9, HasModifiers: true, Invoke: func(execution *Execution, caller TypeID, _ any, arguments []any) any {
			array := ObjectView[*ReferenceArray](arguments[0], ArrayTypeID(reflectionAccessibleObjectTypeID))
			AccessibleObjectSetAccessibleArrayExecution(execution, caller, array, arguments[1].(bool))
			return nil
		}},
		{Name: "trySetAccessible", Return: PrimitiveBooleanTypeID, Modifiers: 17, HasModifiers: true, Invoke: func(execution *Execution, caller TypeID, receiver any, _ []any) any {
			return AccessibleObjectTrySetAccessibleExecution(execution, caller, ObjectView[JavaAccessibleObject](receiver, reflectionAccessibleObjectTypeID))
		}},
		{Name: "isAccessible", Return: PrimitiveBooleanTypeID, Modifiers: 1, HasModifiers: true, Invoke: func(_ *Execution, _ TypeID, receiver any, _ []any) any {
			return AccessibleObjectIsAccessible(ObjectView[JavaAccessibleObject](receiver, reflectionAccessibleObjectTypeID))
		}},
	}})
	RegisterClassDescriptor(ClassDescriptor{Type: AnnotationTypeID, RuntimeLocation: location, Interface: true, Modifiers: 1537, HasModifiers: true, Methods: []MethodDescriptor{{Name: "annotationType", Return: JavaClassTypeID, Modifiers: 1025, HasModifiers: true, Invoke: func(execution *Execution, _ TypeID, receiver any, _ []any) any {
		return AnnotationTypeExecution(execution, ObjectView[JavaAnnotation](receiver, AnnotationTypeID))
	}}}})
	// JDK21 String.value is private final byte[]. This metadata allows the genuine
	// closed java.base access refusal; no fake backing-storage getter is supplied.
	descriptor := classDescriptor(StringTypeID)
	descriptor.Type = StringTypeID
	descriptor.Modifiers = 17
	descriptor.HasModifiers = true
	descriptor.RuntimeLocation = location
	descriptor.Fields = append(descriptor.Fields, FieldDescriptor{Name: "value", Type: ArrayTypeID(PrimitiveByteTypeID), Final: true, NonPublic: true, Modifiers: 18, HasModifiers: true})
	RegisterClassDescriptor(descriptor)
}
