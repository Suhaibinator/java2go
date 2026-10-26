package stdjava

func init() {
	RegisterJavaType(RecordTypeID, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: RecordTypeID, Modifiers: 1025, HasModifiers: true, RuntimeLocation: ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}, PartialMemberMetadata: true})
}
