package stdjava

func streamFromSource[T any](open func(*Execution) streamBoundSource[T]) Stream[T] {
	if open == nil {
		panic(NewNullPointerException("nil stream source"))
	}
	source := &streamSourceState{}
	opener := open
	source.clearSource = func() { opener = nil }
	bind := func(evaluation *streamEvaluation) *streamBoundSource[T] {
		if existing, ok := evaluation.bindings[source]; ok {
			return existing.(*streamBoundSource[T])
		}
		acquire := opener
		streamReleaseSource(source)
		if acquire == nil {
			panic(NewIllegalStateException("stream source is unavailable"))
		}
		bound := acquire(evaluation.execution)
		if bound.drive == nil || bound.exactSize < -1 {
			panic(NewIllegalArgumentException("invalid stream source binding"))
		}
		evaluation.bindings[source] = &bound
		return &bound
	}
	node := &streamNode[T]{state: &streamStageState{source: source}}
	node.exactSize = func(evaluation *streamEvaluation) (int64, bool) {
		bound := bind(evaluation)
		return bound.exactSize, bound.exactSize >= 0
	}
	node.drive = func(evaluation *streamEvaluation, sink streamSink[T]) {
		bound := bind(evaluation)
		sink.Begin(bound.exactSize)
		bound.drive(evaluation, sink)
		sink.End()
	}
	return Stream[T]{pipeline: node}
}

func streamFromSlice[T any](elements []T) Stream[T] {
	owned := append([]T(nil), elements...)
	return streamFromSource(func(*Execution) streamBoundSource[T] {
		return streamBoundSource[T]{int64(len(owned)), func(_ *streamEvaluation, sink streamSink[T]) {
			for _, element := range owned {
				if sink.CancellationRequested() {
					return
				}
				sink.Accept(element)
			}
		}}
	})
}

// A Slice-only custom Iterable cannot support safe lazy traversal/cancellation:
// refuse it explicitly rather than prefetch its callbackful Slice. JavaIterable
// provides a pull protocol, and is used even when Slice is also implemented.
func streamFromIterable[T any](source Iterable[T]) Stream[T] {
	ReferenceRequireNonNull(source)
	if javaSource, ok := source.(JavaIterable); ok {
		return streamFromJavaCollection[T](javaSource)
	}
	if slice, ok := source.(*sliceIterable[T]); ok {
		return streamFromSource(func(*Execution) streamBoundSource[T] {
			return streamBoundSource[T]{int64(len(slice.elements)), func(_ *streamEvaluation, sink streamSink[T]) {
				for _, element := range slice.elements {
					if sink.CancellationRequested() {
						return
					}
					sink.Accept(element)
				}
			}}
		})
	}
	panic(NewUnsupportedOperationException("deferred Stream requires JavaIterable for a custom Iterable"))
}

func streamFromJavaCollection[T any](source JavaIterable) Stream[T] {
	ReferenceRequireNonNull(source)
	if native, ok := source.(*List[T]); ok {
		return streamFromNativeList(native)
	}
	return streamFromSource(func(e *Execution) streamBoundSource[T] {
		size := int64(-1)
		var nativeCursor JavaIterator
		// Do not dispatch a user-defined size method or walk an iterator to learn
		// its size. Only these concrete native collection types are trusted.
		switch native := source.(type) {
		case *Set[T]:
			size = int64(native.Size())
			nativeCursor = native.IteratorJava2goExecution(e)
		}
		// Delay iterator creation until actual traversal. Known-size count does
		// not need an iterator, and limit(0) need not invoke a custom factory.
		return streamBoundSource[T]{size, func(evaluation *streamEvaluation, sink streamSink[T]) {
			if sink.CancellationRequested() {
				return
			}
			cursor := nativeCursor
			if cursor == nil {
				cursor = IterableIteratorExecution(e, source)
			}
			for !sink.CancellationRequested() && IteratorHasNextExecution(e, cursor) {
				sink.Accept(collectionElementView[T](IteratorNextExecution(e, cursor)))
			}
		}}
	})
}

// SubList.spliterator checks the view when the stream is constructed, then
// binds the view's own revision and size later without rechecking the root.
func streamFromNativeList[T any](source *List[T]) Stream[T] {
	source.checkViewModification()
	return streamFromSource(func(*Execution) streamBoundSource[T] {
		return streamBoundListSource(source)
	})
}

// ArrayList's spliterator binds a fence/revision during size acquisition. Bulk
// traversal checks revision after the bound prefix; short-circuit traversal
// checks after each accepted slot. An Iterator's pre-next revision check would
// incorrectly erase callback effects when a collector supplier mutates the list.
func streamBoundListSource[T any](source *List[T]) streamBoundSource[T] {
	size := source.uncheckedSize()
	root := source.storageRoot()
	offset := int32(0)
	if source.viewRoot != nil {
		offset = source.viewOffset
	}
	fence := int64(offset) + int64(size)
	expected := source.modCount
	check := func() {
		if root.modCount != expected {
			panic(NewConcurrentModificationException(""))
		}
	}
	return streamBoundSource[T]{int64(size), func(_ *streamEvaluation, sink streamSink[T]) {
		shortCircuit := streamSinkShortCircuit(sink)
		backing := streamCaptureListBacking(root) // after Begin, before first bulk slot
		for index := int64(offset); index < fence; index++ {
			if sink.CancellationRequested() {
				return
			}
			if shortCircuit {
				backing = streamCaptureListBacking(root) // once per tryAdvance call
			}
			sink.Accept(backing.element(int32(index)))
			if shortCircuit {
				check()
			}
		}
		if !shortCircuit {
			check()
		}
	}}
}

// A captured slice header shares slot mutations but stays with the original
// backing allocation when an append detaches the root during a callback.
type streamListBacking[T any] struct {
	elements       []T
	erasedElements []any
	array          *ReferenceArray
	erased         bool
}

func streamCaptureListBacking[T any](root *List[T]) streamListBacking[T] {
	return streamListBacking[T]{root.elements, root.erasedElements, root.array, root.erasedStorage}
}

func (backing streamListBacking[T]) element(index int32) T {
	var empty T
	if backing.array != nil {
		if index >= ReferenceArrayLength(backing.array) {
			return empty
		}
		return ReferenceArrayGet[T](backing.array, index, ObjectTypeID)
	}
	if backing.erased {
		if int64(index) >= int64(len(backing.erasedElements)) {
			return empty
		}
		return collectionElementView[T](backing.erasedElements[index])
	}
	if int64(index) >= int64(len(backing.elements)) {
		return empty
	}
	return backing.elements[index]
}

func streamFromArray[T any](array any) Stream[T] {
	ReferenceRequireNonNull(array)
	switch values := array.(type) {
	case *PrimitiveArray[T]:
		return streamFromSource(func(*Execution) streamBoundSource[T] {
			return streamBoundSource[T]{int64(len(values.Elements)), func(_ *streamEvaluation, sink streamSink[T]) {
				for index := 0; index < len(values.Elements); index++ {
					if sink.CancellationRequested() {
						return
					}
					sink.Accept(values.Elements[index])
				}
			}}
		})
	case *ReferenceArray:
		return streamFromSource(func(*Execution) streamBoundSource[T] {
			return streamBoundSource[T]{int64(ReferenceArrayLength(values)), func(_ *streamEvaluation, sink streamSink[T]) {
				for index := int32(0); index < ReferenceArrayLength(values); index++ {
					if sink.CancellationRequested() {
						return
					}
					sink.Accept(ReferenceArrayGet[T](values, index, ObjectTypeID))
				}
			}}
		})
	case []T:
		return streamFromSource(func(*Execution) streamBoundSource[T] {
			return streamBoundSource[T]{int64(len(values)), func(_ *streamEvaluation, sink streamSink[T]) {
				for index := range values {
					if sink.CancellationRequested() {
						return
					}
					sink.Accept(values[index])
				}
			}}
		})
	}
	panic(NewIllegalArgumentException("deferred Arrays.stream requires an array"))
}
