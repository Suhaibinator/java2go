package stdjava

import (
	"slices"
	"strings"
	"unicode/utf16"
)

// InvalidPathException retains the provider's String input before any lossy
// native encoding. Its Throwable allocation owns the usual cause state.
type InvalidPathException struct {
	ThrowableBase
	input  *JavaString
	reason *JavaString
	index  int32
}

func (failure InvalidPathException) JavaDynamicTypeID() TypeID {
	return "java.nio.file.InvalidPathException"
}
func (failure InvalidPathException) GetInput() *JavaString  { return failure.input }
func (failure InvalidPathException) GetReason() *JavaString { return failure.reason }
func (failure InvalidPathException) GetIndex() int32        { return failure.index }

func newJavaInvalidPathException(input *JavaString, reason string) InvalidPathException {
	reasonText := JavaStringLiteralUTF16(utf16.Encode([]rune(reason)))
	message := append(reasonText.UTF16Copy(), ':', ' ')
	message = append(message, input.units...)
	return InvalidPathException{
		ThrowableBase: newJavaThrowableBase("InvalidPathException", NewJavaStringUTF16(message)),
		input:         input, reason: reasonText, index: -1,
	}
}

func pathReferenceRequireNonNull(value *JavaString, message *JavaString) {
	if value == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", message)})
	}
}

// PathsGetReference implements the String varargs declaration. It checks every
// segment during assembly, before the Unix provider validates the full input.
func PathsGetReference(first *JavaString, more ...*JavaString) *JavaPath {
	pathReferenceRequireNonNull(first, nil)
	input := first
	if len(more) != 0 {
		units := first.UTF16Copy()
		for _, segment := range more {
			pathReferenceRequireNonNull(segment, JavaStringLiteralUTF16(utf16.Encode([]rune("Cannot invoke \"String.isEmpty()\" because \"segment\" is null"))))
			if len(segment.units) == 0 {
				continue
			}
			if len(units) != 0 {
				units = append(units, '/')
			}
			units = append(units, segment.units...)
		}
		input = NewJavaStringUTF16(units)
	}
	return pathReferenceFromInput(input)
}

// PathsGetArrayReference preserves the distinction between an absent varargs
// array and an empty array, including first-null's precedence over array-null.
func PathsGetArrayReference(first *JavaString, more *ReferenceArray) *JavaPath {
	pathReferenceRequireNonNull(first, nil)
	if more == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", JavaStringLiteralUTF16(utf16.Encode([]rune("Cannot read the array length because \"more\" is null"))))})
	}
	parts := make([]*JavaString, ReferenceArrayLength(more))
	for index := range parts {
		parts[index] = ReferenceArrayGet[*JavaString](more, index, StringTypeID)
	}
	return PathsGetReference(first, parts...)
}

func pathReferenceFromInput(input *JavaString) *JavaPath {
	pathReferenceRequireNonNull(input, nil)
	for _, unit := range input.units {
		if unit == 0 {
			panic(newJavaInvalidPathException(input, "Nul character not allowed"))
		}
	}
	// The provider collapses separators without resolving dot components.
	units := make([]uint16, 0, len(input.units))
	for _, unit := range input.units {
		if unit == '/' && len(units) != 0 && units[len(units)-1] == '/' {
			continue
		}
		units = append(units, unit)
	}
	if len(units) > 1 && units[len(units)-1] == '/' {
		units = units[:len(units)-1]
	}
	if !slices.Equal(units, input.units) {
		input = NewJavaStringUTF16(units)
	}
	// Reject unpaired surrogates before converting the validated pathname to
	// native UTF8 storage. UTF16 decoding must not replace an invalid input.
	for index := 0; index < len(units); index++ {
		unit := units[index]
		if unit >= 0xd800 && unit <= 0xdbff {
			if index+1 >= len(units) || units[index+1] < 0xdc00 || units[index+1] > 0xdfff {
				panic(newJavaInvalidPathException(input, "Malformed input or input contains unmappable characters"))
			}
			index++
		} else if unit >= 0xdc00 && unit <= 0xdfff {
			panic(newJavaInvalidPathException(input, "Malformed input or input contains unmappable characters"))
		}
	}
	return &JavaPath{path: string(utf16.Decode(units))}
}

// PathToStringReference is the Java display boundary. The native ToString API
// remains available to Go callers and filesystem plumbing.
func PathToStringReference(path *JavaPath) *JavaString {
	if path == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	return NewJavaStringUTF16(utf16.Encode([]rune(path.path)))
}

func PathResolveStringReference(path *JavaPath, other *JavaString) *JavaPath {
	if path == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	return PathResolvePathReference(path, PathsGetReference(other))
}

func PathResolvePathReference(path, other *JavaPath) *JavaPath {
	if path == nil || other == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	if strings.HasPrefix(other.path, "/") {
		return other
	}
	if other.path == "" {
		return &JavaPath{path: path.path}
	}
	if path.path == "" {
		return &JavaPath{path: other.path}
	}
	separator := "/"
	if path.path == "/" {
		separator = ""
	}
	return &JavaPath{path: path.path + separator + other.path}
}

func PathIsAbsoluteReference(path *JavaPath) bool {
	if path == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	return path.isAbsolute()
}

// JavaPath models this runtime's single Unix filesystem. Foreign objects and
// null never gain Path equality by rendering the same native text.
func (path *JavaPath) Equals(other any) bool {
	if path == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	other = collectionObjectView(other)
	right, ok := other.(*JavaPath)
	return ok && right != nil && path.path == right.path
}

func (*JavaPath) JavaDynamicTypeID() TypeID { return "sun.nio.fs.UnixPath" }

// Keep the equals/hashCode contract for a Path used through Object or as a
// collection key. UnixPath hashes its encoded pathname bytes.
func (path *JavaPath) HashCode() int32 {
	if path == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	var hash int32
	for index := 0; index < len(path.path); index++ {
		hash = 31*hash + int32(path.path[index])
	}
	return hash
}

func init() {
	// The compiler requests modeled Throwable views through this descriptor.
	// Add only the new canonical owner; source-qualified names retain their IDs.
	builtinThrowableDescriptors["InvalidPathException"] = struct {
		id     TypeID
		parent string
	}{id: "java.nio.file.InvalidPathException", parent: "IllegalArgumentException"}
	RegisterException("InvalidPathException", "IllegalArgumentException")
	RegisterJavaType("java.nio.file.InvalidPathException", BuiltinThrowableTypeID("IllegalArgumentException"))
	RegisterJavaType("java.nio.file.Path", ObjectTypeID)
	RegisterJavaType("sun.nio.fs.UnixPath", ObjectTypeID, "java.nio.file.Path")
}
