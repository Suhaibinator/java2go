package stdjava

import "math"

// Collection keys use Java wrapper value equality without replacing the
// original object retained by the collection's iteration view.
type boxedCollectionKey struct {
	kind TypeID
	bits uint64
}

type nullCollectionKey struct{}

func wrapperCollectionKey(value any) (boxedCollectionKey, bool) {
	switch value := value.(type) {
	case *Boolean:
		var bits uint64
		if value.BooleanValue() {
			bits = 1
		}
		return boxedCollectionKey{BooleanTypeID, bits}, true
	case *Byte:
		return boxedCollectionKey{ByteTypeID, uint64(value.ByteValue())}, true
	case *Short:
		return boxedCollectionKey{ShortTypeID, uint64(value.ShortValue())}, true
	case *Character:
		return boxedCollectionKey{CharacterTypeID, uint64(value.CharValue())}, true
	case *Integer:
		return boxedCollectionKey{IntegerTypeID, uint64(value.IntValue())}, true
	case *Long:
		return boxedCollectionKey{LongTypeID, uint64(value.LongValue())}, true
	case *Float:
		return boxedCollectionKey{FloatTypeID, uint64(math.Float32bits(canonicalNaN32(value.FloatValue())))}, true
	case *Double:
		return boxedCollectionKey{DoubleTypeID, math.Float64bits(canonicalNaN64(value.DoubleValue()))}, true
	default:
		return boxedCollectionKey{}, false
	}
}

func collectionKey(value any) any {
	if javaReferenceIsNull(value) {
		return nullCollectionKey{}
	}
	if key, boxed := wrapperCollectionKey(value); boxed {
		return key
	}
	return value
}
