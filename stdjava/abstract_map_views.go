package stdjava

// AbstractMapDefaultsState belongs to the direct source AbstractMap root.
// Descendants share this state; it contains views only, never map storage.
type AbstractMapDefaultsState struct {
	keys   JavaIterable
	values JavaIterable
}

func AbstractMapKeySetExecution(execution *Execution, source JavaMap, state *AbstractMapDefaultsState) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	if state.keys == nil {
		state.keys = &abstractMapKeySetView{abstractMapCollectionView: abstractMapCollectionView{source: source, keys: true}}
	}
	return state.keys
}
func AbstractMapValuesExecution(execution *Execution, source JavaMap, state *AbstractMapDefaultsState) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	if state.values == nil {
		state.values = &abstractMapCollectionView{source: source}
	}
	return state.values
}

type abstractMapCollectionView struct {
	source JavaMap
	keys   bool
}

func (view *abstractMapCollectionView) nativeJavaInterfaces() []TypeID {
	return []TypeID{CollectionTypeID, IterableTypeID}
}
func (view *abstractMapCollectionView) IteratorJava2goExecution(execution *Execution) JavaIterator {
	return &abstractMapViewIterator{cursor: abstractMapIterator(execution, view.source), keys: view.keys}
}
func (view *abstractMapCollectionView) CollectionSizeJava2goExecution(execution *Execution) int32 {
	return MapSizeExecution(execution, view.source)
}
func (view *abstractMapCollectionView) CollectionIsEmptyJava2goExecution(execution *Execution) bool {
	return MapIsEmptyExecution(execution, view.source)
}
func (view *abstractMapCollectionView) CollectionContainsJava2goExecution(execution *Execution, value any) bool {
	if view.keys {
		return MapContainsKeyExecution(execution, view.source, value)
	}
	return MapContainsValueExecution(execution, view.source, value)
}
func (view *abstractMapCollectionView) CollectionRemoveJava2goExecution(execution *Execution, value any) bool {
	return AbstractCollectionRemoveExecution(execution, view, value)
}
func (view *abstractMapCollectionView) CollectionClearJava2goExecution(execution *Execution) {
	MapClearExecution(execution, view.source)
}
func (view *abstractMapCollectionView) CollectionContainsAllJava2goExecution(execution *Execution, other JavaIterable) bool {
	return AbstractCollectionContainsAllExecution(execution, view, other)
}

type abstractMapKeySetView struct{ abstractMapCollectionView }

func (view *abstractMapKeySetView) nativeJavaInterfaces() []TypeID {
	return []TypeID{SetTypeID, CollectionTypeID, IterableTypeID}
}
func (view *abstractMapKeySetView) EqualsJava2goExecution(execution *Execution, other any) bool {
	return AbstractSetEqualsExecution(execution, view, other)
}
func (view *abstractMapKeySetView) HashCodeJava2goExecution(execution *Execution) int32 {
	return AbstractSetHashCodeExecution(execution, view)
}

type abstractMapViewIterator struct {
	cursor JavaIterator
	keys   bool
}

func (cursor *abstractMapViewIterator) HasNextJava2goExecution(execution *Execution) bool {
	return IteratorHasNextExecution(execution, cursor.cursor)
}
func (cursor *abstractMapViewIterator) NextJava2goExecution(execution *Execution) any {
	entry := abstractMapNextEntry(execution, cursor.cursor)
	if cursor.keys {
		return MapEntryGetKeyExecution(execution, entry)
	}
	return MapEntryGetValueExecution(execution, entry)
}
func (cursor *abstractMapViewIterator) IteratorRemoveJava2goExecution(execution *Execution) {
	IteratorRemoveExecution(execution, cursor.cursor)
}
