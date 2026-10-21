package stdjava

import "fmt"

// Both final Java classes retain one physical ABI. The constructor records
// which canonical declaration allocated it; the Go type cannot prove that.
type stringBuilderClass uint8

const (
	stringBuilderClassBuilder stringBuilderClass = iota + 1
	stringBuilderClassBuffer
	stringBuilderTypeID         TypeID = "java.lang.StringBuilder"
	stringBufferTypeID          TypeID = "java.lang.StringBuffer"
	abstractStringBuilderTypeID TypeID = "java.lang.AbstractStringBuilder"
	appendableTypeID            TypeID = "java.lang.Appendable"
)

func init() {
	RegisterJavaType(appendableTypeID, ObjectTypeID)
	RegisterJavaType(abstractStringBuilderTypeID, ObjectTypeID, appendableTypeID, CharSequenceTypeID)
	for _, id := range []TypeID{stringBuilderTypeID, stringBufferTypeID} {
		RegisterJavaType(id, abstractStringBuilderTypeID, appendableTypeID, SerializableTypeID, ComparableTypeID, CharSequenceTypeID)
		RegisterClassDescriptor(ClassDescriptor{Type: id, HasModifiers: true, Modifiers: 17})
	}
	// This is the real package-private abstract superclass, never a runtime
	// allocation identity or an instantiable shared replacement class.
	RegisterClassDescriptor(ClassDescriptor{Type: abstractStringBuilderTypeID, HasModifiers: true, Modifiers: 1024})
	for _, id := range []TypeID{appendableTypeID, SerializableTypeID, ComparableTypeID, CharSequenceTypeID} {
		RegisterClassDescriptor(ClassDescriptor{Type: id, Interface: true, HasModifiers: true, Modifiers: 1537})
	}
}

func newStringBuilderAllocation(class stringBuilderClass, units []rune) *StringBuilder {
	return &StringBuilder{buf: units, class: class}
}

func (b *StringBuilder) JavaDynamicTypeID() TypeID {
	if b == nil {
		return ""
	}
	switch b.class {
	case stringBuilderClassBuilder:
		return stringBuilderTypeID
	case stringBuilderClassBuffer:
		return stringBufferTypeID
	}
	return ""
}

// NewStringBuffer allocates the canonical Buffer class using the shared ABI.
func NewStringBuffer() *StringBuilder {
	return newStringBuilderAllocation(stringBuilderClassBuffer, nil)
}

func NewStringBufferCapacity(capacity int32) *StringBuilder {
	if capacity < 0 {
		panic(NewNegativeArraySizeException(fmt.Sprint(capacity)))
	}
	return newStringBuilderAllocation(stringBuilderClassBuffer, make([]rune, 0, int(capacity)))
}
