package stdjava

import (
	"strings"
	"sync"
	"unicode/utf16"
)

var javaClassNameReferences sync.Map

// ClassJavaName exposes the native class descriptor's name as a canonical Java
// String. Repeated reads of a class name retain the same immutable reference.
func ClassJavaName(class *Class) *JavaString {
	ReferenceRequireNonNull(class)
	id := class.TypeID()
	if name, found := javaClassNameReferences.Load(id); found {
		return name.(*JavaString)
	}
	// Class binary names are canonical JVM metadata identifiers, including array descriptors.
	// Seed the literal pool so reads share the pointer returned by matching literals.
	name := JavaStringLiteralUTF16(utf16.Encode([]rune(javaClassBinaryName(id))))
	stored, _ := javaClassNameReferences.LoadOrStore(id, name)
	return stored.(*JavaString)
}

// Array class names use JVM descriptors, while ordinary and primitive classes
// expose their binary names. Internal reified-array TypeIDs stay unchanged.
func javaClassBinaryName(id TypeID) string {
	if component, array := arrayComponentTypeID(id); array {
		if _, nested := arrayComponentTypeID(component); nested {
			return "[" + javaClassBinaryName(component)
		}
		if isPrimitiveTypeID(component) {
			descriptors := map[string]string{
				"boolean": "Z", "byte": "B", "char": "C", "short": "S",
				"int": "I", "long": "J", "float": "F", "double": "D",
			}
			if descriptor, found := descriptors[strings.TrimPrefix(string(component), primitiveTypePrefix)]; found {
				return "[" + descriptor
			}
			panic(NewIllegalArgumentException("invalid primitive array class"))
		}
		return "[L" + string(component) + ";"
	}
	return strings.TrimPrefix(string(id), primitiveTypePrefix)
}

var javaClassSimpleNameReferences sync.Map

// ClassJavaSimpleName exposes the descriptor-derived Java simple name. Its
// per-TypeID cache preserves identity without interning unrelated classes.
func ClassJavaSimpleName(class *Class) *JavaString {
	ReferenceRequireNonNull(class)
	id := class.TypeID()
	if name, found := javaClassSimpleNameReferences.Load(id); found {
		return name.(*JavaString)
	}
	name := NewJavaStringUTF16(utf16.Encode([]rune(class.GetSimpleName())))
	stored, _ := javaClassSimpleNameReferences.LoadOrStore(id, name)
	return stored.(*JavaString)
}

func (class *Class) GetNameJavaString() *JavaString       { return ClassJavaName(class) }
func (class *Class) GetSimpleNameJavaString() *JavaString { return ClassJavaSimpleName(class) }
