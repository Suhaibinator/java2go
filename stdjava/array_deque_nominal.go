package stdjava

const ArrayDequeTypeID TypeID = "java.util.ArrayDeque"
const DequeTypeID TypeID = "java.util.Deque"
const QueueTypeID TypeID = "java.util.Queue"

func init() {
	fact := listDeclarationFact{ArrayDequeTypeID, AbstractCollectionTypeID, []TypeID{DequeTypeID, CloneableTypeID, SerializableTypeID}, false, 1}
	RegisterJavaType(fact.id, fact.super, fact.interfaces...)
	reflectionRegistry.Store(fact.id, listDeclarationDescriptor(classDescriptor(fact.id), fact))
	RegisterClassRuntimeLocation(fact.id, ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"})
}
func (*ArrayDeque) JavaDynamicTypeID() TypeID      { return ArrayDequeTypeID }
func (*ArrayDeque) nativeJavaInterfaces() []TypeID { return []TypeID{DequeTypeID} }
