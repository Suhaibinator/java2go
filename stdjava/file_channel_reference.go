package stdjava

import "sync/atomic"

// FileChannelSourceMethods are declaration-selected source bodies. Generated
// callers pass their current Execution to the virtual facades; these immutable
// slots never capture a constructor's execution or inspect host method shapes.
type FileChannelSourceMethods struct {
	Read      func(*Execution, *ByteBuffer) int32
	ReadAt    func(*Execution, *ByteBuffer, int64) int32
	ImplClose func(*Execution)
}

type fileChannelSourceState struct {
	rootInfo  *ObjectInfo
	methods   FileChannelSourceMethods
	closeLock any
	closed    atomic.Bool
}

func fileChannelNativeRootID(id TypeID) bool {
	// These are exact canonical platform declarations, not simple-name tests.
	// Rejection does not register or fabricate FileChannelImpl value metadata.
	return id == JavaNIOFileChannelType || id == TypeID("sun.nio.ch.FileChannelImpl")
}

// NewFileChannelSourceReference creates the single canonical view of a
// compiler-owned source allocation. It performs no Java callback or native I/O.
// Failed certificates describe an internal construction invariant, not a Java
// FileChannel constructor contract. Compiler source registration precedes this
// call, and the supplied ObjectInfo belongs to the most-derived source root.
func NewFileChannelSourceReference(rootInfo *ObjectInfo, methods FileChannelSourceMethods) *FileChannel {
	if rootInfo == nil || fileChannelNativeRootID(rootInfo.DynamicType()) || !registeredJavaSourceValue(rootInfo) {
		panic(NewUnsupportedOperationException("FileChannel source root is not a registered source declaration"))
	}
	// The existing source predicate has released the registry lock before this
	// independent ancestry query. The ObjectInfo view provider is never invoked.
	if !JavaTypeAssignable(rootInfo.DynamicType(), JavaNIOFileChannelType) || methods.Read == nil || methods.ReadAt == nil || methods.ImplClose == nil {
		panic(NewUnsupportedOperationException("FileChannel source declaration has an incomplete protocol"))
	}
	return &FileChannel{source: &fileChannelSourceState{rootInfo: rootInfo, methods: methods, closeLock: NewObject()}}
}

func fileChannelSourceBacking(channel *FileChannel) *fileChannelSourceState {
	ReferenceRequireNonNull(channel)
	if (channel.source == nil) == (channel.state == nil) {
		panic(NewUnsupportedOperationException("FileChannel has no unique source or native backing"))
	}
	return channel.source
}

// JavaObjectInfo keeps source/root/base views at one Java allocation identity.
// Existing native RAF channels remain opaque; nil is not abstract-class metadata.
func (channel *FileChannel) JavaObjectInfo() *ObjectInfo {
	if channel == nil || channel.source == nil {
		return nil
	}
	return channel.source.rootInfo
}

func FileChannelReadVirtualExecution(execution *Execution, channel *FileChannel, dst *ByteBuffer) int32 {
	requireExecution(execution)
	if source := fileChannelSourceBacking(channel); source != nil {
		// Java source overrides own validation/effects, even for closed, interrupted,
		// readonly or null arguments. No native state/descriptor mutex is acquired.
		return source.methods.Read(execution, dst)
	}
	return FileChannelReadExecution(execution, channel, dst)
}

func FileChannelReadAtVirtualExecution(execution *Execution, channel *FileChannel, dst *ByteBuffer, position int64) int32 {
	requireExecution(execution)
	if source := fileChannelSourceBacking(channel); source != nil {
		return source.methods.ReadAt(execution, dst, position)
	}
	return fileChannelReadAtNativeExecution(execution, channel, dst, position)
}

func FileChannelCloseVirtualExecution(execution *Execution, channel *FileChannel) {
	requireExecution(execution)
	if source := fileChannelSourceBacking(channel); source != nil {
		// AbstractInterruptibleChannel always enters its private reentrant monitor,
		// including an already-closed call. Concurrent closers wait for implClose;
		// isOpen observes closed without waiting on this monitor.
		guard := MonitorEnterExecution(execution, source.closeLock)
		defer MonitorExitExecution(guard)
		if source.closed.Load() {
			return
		}
		source.closed.Store(true)
		source.methods.ImplClose(execution)
		return
	}
	channel.state.close()
}

func FileChannelIsOpenVirtual(channel *FileChannel) bool {
	if source := fileChannelSourceBacking(channel); source != nil {
		return !source.closed.Load()
	}
	channel.state.mu.Lock()
	defer channel.state.mu.Unlock()
	return !channel.state.closed
}
