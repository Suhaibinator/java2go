package stdjava

func init() {
	location := ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}
	RegisterJavaType(RecordComponentTypeID, ObjectTypeID, AnnotatedElementTypeID)
	descriptor := classDescriptor(JavaClassTypeID)
	descriptor.Type = JavaClassTypeID
	descriptor.RuntimeLocation = location
	descriptor.Modifiers = 17
	descriptor.HasModifiers = true
	descriptor.PartialMemberMetadata = true
	descriptor.Methods = append(descriptor.Methods,
		MethodDescriptor{Name: "isRecord", Return: PrimitiveBooleanTypeID, Modifiers: 1, HasModifiers: true, Invoke: func(_ *Execution, _ TypeID, receiver any, _ []any) any {
			return ObjectView[*Class](receiver, JavaClassTypeID).IsRecord()
		}},
		MethodDescriptor{Name: "getRecordComponents", Return: ArrayTypeID(RecordComponentTypeID), Modifiers: 1, HasModifiers: true, Invoke: func(_ *Execution, _ TypeID, receiver any, _ []any) any {
			return ObjectView[*Class](receiver, JavaClassTypeID).GetRecordComponents()
		}},
	)
	RegisterClassDescriptor(descriptor)
	RegisterClassDescriptor(ClassDescriptor{Type: RecordComponentTypeID, RuntimeLocation: location, Modifiers: 17, HasModifiers: true, PartialMemberMetadata: true, Methods: []MethodDescriptor{
		{Name: "getName", Return: StringTypeID, Modifiers: 1, HasModifiers: true, Invoke: func(_ *Execution, _ TypeID, receiver any, _ []any) any {
			return ObjectView[*RecordComponent](receiver, RecordComponentTypeID).GetNameJavaString()
		}},
		{Name: "getType", Return: JavaClassTypeID, Modifiers: 1, HasModifiers: true, Invoke: func(_ *Execution, _ TypeID, receiver any, _ []any) any {
			return ObjectView[*RecordComponent](receiver, RecordComponentTypeID).GetType()
		}},
	}})
}
