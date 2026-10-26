package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"strings"
)

const (
	fileChannelSourceStorage     = "__java2goFileChannel"
	fileChannelSourceReadSlot    = "Java2goFileChannelReadExecution"
	fileChannelSourceReadAtSlot  = "Java2goFileChannelReadAtExecution"
	fileChannelSourceCloseSlot   = "Java2goFileChannelImplCloseExecution"
	fileChannelSourceWriteSlot   = "Java2goFileChannelWriteExecution"
	fileChannelSourceWriteAtSlot = "Java2goFileChannelWriteAtExecution"
)

func fileChannelSourceRoot(scope *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	if scope == nil || scope.IsInterface || !fileChannelSourceHasCanonicalRoot(scope, ctx) {
		return nil
	}
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		parent := resolveSuperclassScopeInDeclaringContext(ctx, scope)
		if parent == nil {
			return scope
		}
		scope = parent
	}
	return nil
}
func fileChannelSourceDirectRoot(scope *symbol.ClassScope, ctx Ctx) bool {
	return scope != nil && fileChannelSourceRoot(scope, ctx) == scope
}

func fileChannelSourceByteBufferParameter(parameter *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if parameter == nil || parameter.DirectTypeParameter != nil || findFileScopeForClassScope(owner, ctx) == nil {
		return false
	}
	declaring := ctx.Clone()
	declaring.currentFile = nil
	declaring.syntheticTypeParameters = nil
	declaring.localClasses = nil
	declaring = classScopeCtx(owner, declaring)
	base, rank := javaArrayTypeParts(strings.TrimSpace(parameter.OriginalType))
	base, args := parseJavaTypeString(base)
	first := strings.Split(base, ".")[0]
	if rank != 0 || len(args) != 0 || parameter.TypeParameterBindings[base] != nil || visibleTypeParameterDeclarationForJavaType(first, declaring) != nil || resolveClassScopeByQualifiedName(declaring, base) != nil || resolveClassScopeByQualifiedName(declaring, first) != nil {
		return false
	}
	if base == "java.nio.ByteBuffer" {
		return true
	}
	if base != "ByteBuffer" || declaring.currentFile == nil {
		return false
	}
	if imported, present := declaring.currentFile.Imports[base]; present {
		return imported == "java.nio"
	}
	for _, owner := range ordinaryTypeOnDemandImports(declaring) {
		if owner == "java.nio" {
			return true
		}
	}
	return false
}

// Fixed slots describe actual Java declarations, never matching Go method sets.
func fileChannelSourceProtocolSlot(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	if method == nil || method.Constructor || method.IsStatic || method.IsPrivate || len(method.TypeParameters) != 0 {
		return ""
	}
	if method.OriginalName == "implCloseChannel" {
		if len(method.Parameters) != 0 || method.OriginalType != "void" || fileChannelSourceRoot(owner, ctx) == nil {
			return ""
		}
		return fileChannelSourceCloseSlot
	}
	if (method.OriginalName != "read" && method.OriginalName != "write") || method.OriginalType != "int" || len(method.Parameters) < 1 || len(method.Parameters) > 2 || executionParameterIsVariadic(method, len(method.Parameters)-1) || method.Parameters[0] == nil {
		return ""
	}
	// Reject only known incompatible descriptors here. A nil second parameter
	// retains the original ancestry-guarded final check below, so rejected
	// owners cannot acquire a new descriptor dereference panic.
	if len(method.Parameters) == 2 && method.Parameters[1] != nil && (method.Parameters[1].OriginalType != "long" || method.Parameters[1].DirectTypeParameter != nil) {
		return ""
	}
	if fileChannelSourceRoot(owner, ctx) == nil || !fileChannelSourceByteBufferParameter(method.Parameters[0], owner, ctx) {
		return ""
	}
	relativeSlot, positionalSlot := fileChannelSourceReadSlot, fileChannelSourceReadAtSlot
	if method.OriginalName == "write" {
		relativeSlot, positionalSlot = fileChannelSourceWriteSlot, fileChannelSourceWriteAtSlot
	}
	if len(method.Parameters) == 1 {
		return relativeSlot
	}
	if len(method.Parameters) == 2 && method.Parameters[1].OriginalType == "long" && method.Parameters[1].DirectTypeParameter == nil {
		return positionalSlot
	}
	return ""
}

func fileChannelSourceReservedSelector(scope *symbol.ClassScope, name string, ctx Ctx) bool {
	switch name {
	case fileChannelSourceStorage, fileChannelSourceViewMethod, fileChannelSourceReadSlot, fileChannelSourceReadAtSlot, fileChannelSourceCloseSlot, fileChannelSourceWriteSlot, fileChannelSourceWriteAtSlot:
		return fileChannelSourceRoot(scope, ctx) != nil
	}
	return false
}
func fileChannelSourceCanonicalViewIDs() []string {
	return []string{"java.nio.channels.FileChannel", "java.nio.channels.ReadableByteChannel", "java.nio.channels.WritableByteChannel", "java.nio.channels.Channel", "java.nio.channels.ByteChannel", "java.nio.channels.InterruptibleChannel", "java.nio.channels.spi.AbstractInterruptibleChannel", "java.nio.channels.SeekableByteChannel", "java.nio.channels.GatheringByteChannel", "java.nio.channels.ScatteringByteChannel", "java.io.Closeable", "java.lang.AutoCloseable"}
}
