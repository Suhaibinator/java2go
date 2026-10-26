package stdjava

import "reflect"

// SystemIdentityHashCode returns Object's identity hash regardless of any Java
// hashCode override. Null has hash zero; superclass views share ObjectInfo.
func SystemIdentityHashCode(value any) int32 {
	if javaReferenceIsNull(value) {
		return 0
	}
	return objectIdentityHashCode(collectionObjectView(value))
}

// objectIdentityHashCode is the shared default after virtual Object.hashCode
// dispatch and System.identityHashCode's explicit bypass of that dispatch.
func objectIdentityHashCode(value any) int32 {
	if carrier, ok := value.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil {
			pointer := uint64(reflect.ValueOf(info).Pointer())
			return int32(pointer ^ (pointer >> 32))
		}
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Chan, reflect.Slice, reflect.Func:
		pointer := uint64(reflected.Pointer())
		return int32(pointer ^ (pointer >> 32))
	default:
		panic(NewClassCastException("value has no Java object identity"))
	}
}
