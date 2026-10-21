package stdjava

// Allocation identity belongs to the backing object. Legacy factories and
// views retain the unknown zero value until their concrete wrappers exist.
type listAllocationKind uint8

const (
	listAllocationUnknown listAllocationKind = iota
	listAllocationArray
	listAllocationLinked
)

func (list *List[T]) JavaDynamicTypeID() TypeID {
	if list != nil {
		switch list.allocationKind {
		case listAllocationArray:
			return "java.util.ArrayList"
		case listAllocationLinked:
			return "java.util.LinkedList"
		}
	}
	return ""
}

func NewArrayList[T any]() *List[T] {
	return &List[T]{allocationKind: listAllocationArray}
}
func NewLinkedList[T any]() *List[T] {
	return &List[T]{allocationKind: listAllocationLinked}
}
func NewArrayListWithArgument[T any](argument any, executions ...*Execution) *List[T] {
	switch capacity := argument.(type) {
	case int:
		return newListWithAllocationCapacity[T](int64(capacity), listAllocationArray)
	case int32:
		return newListWithAllocationCapacity[T](int64(capacity), listAllocationArray)
	}
	return newListFromCollection[T](argument, listAllocationArray, executions...)
}
func NewLinkedListFromCollection[T any](argument any, executions ...*Execution) *List[T] {
	return newListFromCollection[T](argument, listAllocationLinked, executions...)
}
func newListWithAllocationCapacity[T any](capacity int64, kind listAllocationKind) *List[T] {
	if capacity < 0 {
		panic(NewIllegalArgumentException("Illegal Capacity"))
	}
	return &List[T]{allocationKind: kind, elements: make([]T, 0, int(capacity))}
}
func newListFromCollection[T any](argument any, kind listAllocationKind, executions ...*Execution) *List[T] {
	ReferenceRequireNonNull(argument)
	if source, ok := argument.(JavaIterable); ok {
		execution := optionalComparisonExecution(executions)
		if execution == nil {
			execution = NewExecution()
		}
		list := &List[T]{allocationKind: kind, erasedStorage: true}
		for _, item := range ErasedCollectionIterationElements(execution, source) {
			list.erasedElements = append(list.erasedElements, item)
		}
		return list
	}
	source := AsIterable[T](argument)
	values := source.Slice()
	copied := make([]T, len(values))
	copy(copied, values)
	return &List[T]{allocationKind: kind, elements: copied}
}

type listDeclarationFact struct {
	id, super   TypeID
	interfaces  []TypeID
	isInterface bool
	modifiers   int32
}

// These JDK21 declarations were observed directly, including direct interfaces.
// Interface graph nodes use the runtime's Object edge convention; reflection
// still returns no superclass for an interface.
var listDeclarationFacts = []listDeclarationFact{
	{"java.util.ArrayList", "java.util.AbstractList", []TypeID{"java.util.List", "java.util.RandomAccess", "java.lang.Cloneable", "java.io.Serializable"}, false, 1},
	{"java.util.LinkedList", "java.util.AbstractSequentialList", []TypeID{"java.util.List", "java.util.Deque", "java.lang.Cloneable", "java.io.Serializable"}, false, 1},
	{"java.util.AbstractList", "java.util.AbstractCollection", []TypeID{"java.util.List"}, false, 1025},
	{"java.util.AbstractSequentialList", "java.util.AbstractList", nil, false, 1025},
	{"java.util.AbstractCollection", ObjectTypeID, []TypeID{"java.util.Collection"}, false, 1025},
	{"java.util.List", ObjectTypeID, []TypeID{"java.util.SequencedCollection"}, true, 1537},
	{"java.util.Collection", ObjectTypeID, []TypeID{"java.lang.Iterable"}, true, 1537},
	{"java.util.Deque", ObjectTypeID, []TypeID{"java.util.Queue", "java.util.SequencedCollection"}, true, 1537},
	{"java.util.Queue", ObjectTypeID, []TypeID{"java.util.Collection"}, true, 1537},
	{"java.util.RandomAccess", ObjectTypeID, nil, true, 1537},
	{"java.util.SequencedCollection", ObjectTypeID, []TypeID{"java.util.Collection"}, true, 1537},
	{"java.lang.Iterable", ObjectTypeID, nil, true, 1537},
	{"java.lang.Cloneable", ObjectTypeID, nil, true, 1537},
	{"java.io.Serializable", ObjectTypeID, nil, true, 1537},
}

// Merge flags into a copy: full members and an existing initialization
// coordinator/callback retain their identity and are never wrapped or invoked.
func listDeclarationDescriptor(descriptor ClassDescriptor, fact listDeclarationFact) ClassDescriptor {
	if descriptor.HasModifiers && (descriptor.Modifiers != fact.modifiers || descriptor.Interface != fact.isInterface) {
		panic("conflicting canonical list declaration metadata")
	}
	descriptor.Interface = fact.isInterface
	descriptor.Modifiers = fact.modifiers
	descriptor.HasModifiers = true
	return descriptor
}
func init() {
	for _, fact := range listDeclarationFacts {
		descriptor := listDeclarationDescriptor(classDescriptor(fact.id), fact)
		RegisterClassRuntimeLocation(fact.id, ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"})
		RegisterJavaType(fact.id, fact.super, fact.interfaces...)
		reflectionRegistry.Store(fact.id, descriptor)
	}
}
