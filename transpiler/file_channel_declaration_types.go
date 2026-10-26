package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// This declaration catalog supplies physical signature types only. It does not
// register constructors, operations, intrinsic owners, or runtime identities.
func fileChannelDeclarationRuntimeTypeExpr(baseName string, ctx Ctx) (ast.Expr, bool) {
	owner, known := fileChannelDeclarationOwner(baseName, ctx)
	if !known {
		return nil, false
	}
	name, pointer := "", true
	switch owner {
	case "java.nio.channels.FileChannel":
		name = "FileChannel"
	case "java.nio.channels.ReadableByteChannel":
		name, pointer = "JavaReadableByteChannel", false
	case "java.nio.channels.WritableByteChannel":
		name, pointer = "JavaWritableByteChannel", false
	case "java.nio.channels.FileChannel.MapMode":
		name = "JavaFileChannelMapMode"
	case "java.nio.MappedByteBuffer":
		name = "JavaMappedByteBuffer"
	case "java.nio.channels.FileLock":
		name = "JavaFileLock"
	default:
		return nil, false
	}
	result := stdjavaQualifiedExpr(name, ctx)
	if pointer {
		return &ast.StarExpr{X: result}, true
	}
	return result, true
}

func fileChannelDeclarationOwner(baseName string, ctx Ctx) (string, bool) {
	base, rank := javaArrayTypeParts(strings.TrimSpace(baseName))
	base, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || base == "" {
		return "", false
	}
	// A simple name filters candidates only; declaration proof below admits them.
	switch stripJavaQualifier(base) {
	case "FileChannel", "ReadableByteChannel", "WritableByteChannel", "MapMode", "MappedByteBuffer", "FileLock":
	default:
		return "", false
	}
	first := strings.Split(base, ".")[0]
	if visibleTypeParameterDeclarationForJavaType(first, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", false
	}
	if base == "MapMode" {
		for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
			if fileChannelSourceHasCanonicalRoot(scope, ctx) {
				return "java.nio.channels.FileChannel.MapMode", true
			}
		}
	}
	if strings.HasSuffix(base, ".MapMode") {
		ownerName := strings.TrimSuffix(base, ".MapMode")
		if owner := resolveClassScopeByQualifiedName(ctx, ownerName); owner != nil && fileChannelSourceHasCanonicalRoot(owner, ctx) {
			return "java.nio.channels.FileChannel.MapMode", true
		}
	}
	return fileChannelDeclarationExternalOwner(base, ctx)
}

func fileChannelDeclarationExternalOwner(baseName string, ctx Ctx) (string, bool) {
	base, rank := javaArrayTypeParts(strings.TrimSpace(baseName))
	base, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || base == "" {
		return "", false
	}
	// A simple name filters candidates only; declaration proof below admits them.
	switch stripJavaQualifier(base) {
	case "FileChannel", "ReadableByteChannel", "WritableByteChannel", "MapMode", "MappedByteBuffer", "FileLock":
	default:
		return "", false
	}
	first := strings.Split(base, ".")[0]
	if visibleTypeParameterDeclarationForJavaType(first, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil || resolveClassScopeByQualifiedName(ctx, first) != nil {
		return "", false
	}
	canonical := func(owner string) bool {
		switch owner {
		case "java.nio.channels.FileChannel", "java.nio.channels.ReadableByteChannel", "java.nio.channels.WritableByteChannel", "java.nio.channels.FileChannel.MapMode", "java.nio.MappedByteBuffer", "java.nio.channels.FileLock":
			return true
		}
		return false
	}
	if canonical(base) {
		return base, true
	}
	if strings.Contains(base, ".") {
		if strings.HasSuffix(base, ".MapMode") {
			owner, known := fileChannelDeclarationExternalOwner(strings.TrimSuffix(base, ".MapMode"), ctx)
			if known && owner == "java.nio.channels.FileChannel" {
				return "java.nio.channels.FileChannel.MapMode", true
			}
		}
		return "", false
	}
	if ctx.currentFile == nil {
		return "", false
	}
	if owner, imported := ctx.currentFile.Imports[base]; imported {
		candidate := owner + "." + base
		return candidate, canonical(candidate)
	}
	candidate := ""
	for _, owner := range ordinaryTypeOnDemandImports(ctx) {
		qualified := owner + "." + base
		if canonical(qualified) {
			if candidate != "" && candidate != qualified {
				return "", false
			}
			candidate = qualified
		}
	}
	return candidate, candidate != ""
}

func fileChannelSourceHasCanonicalRoot(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		// classScopeCtx otherwise retains a caller file when ownership is absent.
		// A detached declaration cannot borrow that file's imports as ancestry proof.
		file := findFileScopeForClassScope(scope, ctx)
		if file == nil {
			return false
		}
		declaring := ctx.Clone()
		declaring.currentFile = nil
		declaring.syntheticTypeParameters = nil
		declaring = classHeaderTypeCtx(scope, declaring)
		if declaring.currentFile != file {
			return false
		}
		parent, args := parseJavaTypeString(strings.TrimSpace(scope.Superclass))
		if parent == "" {
			return false
		}
		if sourceParent := resolveSuperclassScopeInDeclaringContext(declaring, scope); sourceParent != nil {
			scope = sourceParent
			continue
		}
		if len(args) != 0 {
			return false
		}
		owner, known := fileChannelDeclarationExternalOwner(parent, declaring)
		return known && owner == "java.nio.channels.FileChannel"
	}
	return false
}
