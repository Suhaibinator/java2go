package stdjava

// Fixture-only extraction: no toList/grouping/facade production admission.

type collectorToListSourceArray struct {
	info    *ObjectInfo
	want    *Execution
	values  []any
	adds    int
	failure any
}

func (source *collectorToListSourceArray) JavaObjectInfo() *ObjectInfo { return source.info }

func (source *collectorToListSourceArray) JavaDynamicTypeID() TypeID {
	if source.info != nil {
		return source.info.DynamicType()
	}
	return "java.util.ArrayList"
}

func (source *collectorToListSourceArray) IteratorJava2goExecution(e *Execution) JavaIterator {
	if e != source.want {
		panic("source iterator lost terminal Execution")
	}
	return &streamFacadeCursor{source: &streamFacadeIterable{want: e, values: source.values}}
}

func (source *collectorToListSourceArray) CollectionAddJava2goExecution(e *Execution, item any) bool {
	if e != source.want {
		panic("source add lost terminal Execution")
	}
	source.adds++
	if source.failure != nil {
		panic(source.failure)
	}
	source.values = append(source.values, item)
	return true
}

func collectorToListActualSource(e *Execution) *collectorToListSourceArray {
	const id TypeID = "collector.toList.contract.SourceArray"
	RegisterJavaType(id, "java.util.ArrayList")
	RegisterJavaSourceType(id)
	source := &collectorToListSourceArray{want: e}
	base := NewArrayList[any]()
	source.info = NewObjectInfo(id, func(requested TypeID) any {
		if requested == "java.util.ArrayList" || requested == nativeListTypeID {
			return base
		}
		if requested == id || requested == nativeCollectionTypeID || requested == IterableTypeID || requested == ObjectTypeID {
			return source
		}
		return nil
	})
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	return source
}

type collectorToListUnbridgedSourceArray struct {
	*List[any]
	info *ObjectInfo
}

func (source *collectorToListUnbridgedSourceArray) JavaObjectInfo() *ObjectInfo { return source.info }

func collectorToListUnbridgedSource() *collectorToListUnbridgedSourceArray {
	const id TypeID = "collector.toList.contract.UnbridgedSourceArray"
	RegisterJavaType(id, "java.util.ArrayList")
	RegisterJavaSourceType(id)
	source := &collectorToListUnbridgedSourceArray{List: NewArrayList[any]()}
	source.info = NewObjectInfo(id, func(requested TypeID) any {
		if requested == id || requested == "java.util.ArrayList" || requested == nativeListTypeID || requested == nativeCollectionTypeID || requested == IterableTypeID || requested == ObjectTypeID {
			return source
		}
		return nil
	})
	return source
}

type streamFacadeIterable struct {
	values  []any
	want    *Execution
	events  []string
	failure any
}

type streamFacadeCursor struct {
	source *streamFacadeIterable
	index  int
}

func (source *streamFacadeIterable) IteratorJava2goExecution(e *Execution) JavaIterator {
	if e != source.want {
		panic("iterator lost terminal Execution")
	}
	source.events = append(source.events, "iterator")
	return &streamFacadeCursor{source: source}
}

func (cursor *streamFacadeCursor) HasNextJava2goExecution(e *Execution) bool {
	if e != cursor.source.want {
		panic("hasNext lost terminal Execution")
	}
	cursor.source.events = append(cursor.source.events, "hasNext")
	return cursor.index < len(cursor.source.values)
}

func (cursor *streamFacadeCursor) NextJava2goExecution(e *Execution) any {
	if e != cursor.source.want {
		panic("next lost terminal Execution")
	}
	cursor.source.events = append(cursor.source.events, "next")
	if cursor.source.failure != nil {
		panic(cursor.source.failure)
	}
	item := cursor.source.values[cursor.index]
	cursor.index++
	return item
}
