package stdjava

// canonicalObject is the actual java.lang.Object allocation. Nonzero storage
// preserves distinct pointer identity without exposing a mutable type tag.
type canonicalObject struct {
	_ byte
}

func (object *canonicalObject) JavaDynamicTypeID() TypeID {
	if object == nil {
		return ""
	}
	return ObjectTypeID
}
