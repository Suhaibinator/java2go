package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// A declaration certificate and a runtime-owned state copier are separate
// requirements. This closed capability inventory never admits native APIs.
// No native superclass is accepted just because it ultimately extends Object.
type nativeObjectCloneContract struct {
	canonicalOwner, superclass, selectedOwner, descriptor    string
	classSHA, selectedClassSHA, sourceSHA, selectedSourceSHA string
	methodAccess                                             uint16
	declaresClone                                            bool
	stateCopier                                              string
}

// Derived from pinned JDK21.0.6 java.base class/source metadata; full method
// enumeration proves Reader declares no clone. Object owns the protected native
// ()Object body. Runtime representation is ReaderBase{Lock any}; Reader.skip's
// private skipBuffer is unrepresented and that service remains unsupported.
var nativeObjectCloneContracts = map[string]nativeObjectCloneContract{
	"java.io.Reader": {
		canonicalOwner: "java.io.Reader", superclass: "java.lang.Object",
		selectedOwner: "java.lang.Object", descriptor: "()Ljava/lang/Object;", methodAccess: 0x0104,
		classSHA:          "ed61063e20ddf6d7eccb4f1b0a3cf384569784bacb4792f8e00cf1bbe26dd58d",
		selectedClassSHA:  "10cc90d16479279e1d6ee258fdaea4c5640ac6125acb5a7100a7b0a20237a9f2",
		sourceSHA:         "f1a10d791c30c9279d58a41c80842e74340198264f8445f4c46dc351b027b017",
		selectedSourceSHA: "dd95fa54264320f76254ab4edcbe355def1b733cdc967b4972bec0f30f494604",
		stateCopier:       "CloneReaderBaseReference",
	},
}

type nativeObjectCloneCapability struct {
	root            *symbol.ClassScope
	contract        nativeObjectCloneContract
	storageSelector string
}

// Confirm the actual registered declaration and its AST tree before switching
// context. Detached/caller contexts cannot donate native ownership evidence.
func nativeObjectCloneDeclaringContext(scope *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return Ctx{}, false
	}
	file := findFileScopeForClassScope(scope, ctx)
	if file == nil || file.ParsedRoot() == nil {
		return Ctx{}, false
	}
	actual := scope.Class.DeclarationNode
	for actual.Parent() != nil {
		actual = actual.Parent()
	}
	if !actual.Equal(file.ParsedRoot()) {
		return Ctx{}, false
	}
	declaring := ctx.Clone()
	declaring.currentFile = nil
	declaring = classHeaderTypeCtx(scope, declaring)
	if declaring.currentFile != file {
		return Ctx{}, false
	}
	return declaring, true
}

// This certifies the terminal native declaration/storage, not source method
// selection. sourceSuperSelectsObjectClone retains its existing per-source
// zero-argument override checks before consulting the terminal capability.
func sourceNativeObjectCloneCapability(scope *symbol.ClassScope, ctx Ctx) (nativeObjectCloneCapability, bool) {
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil && !seen[current]; {
		seen[current] = true
		if current.IsEnum || current.IsInterface {
			return nativeObjectCloneCapability{}, false
		}
		declaring, owned := nativeObjectCloneDeclaringContext(current, ctx)
		if !owned {
			return nativeObjectCloneCapability{}, false
		}
		if parent := resolveSuperclassScopeInDeclaringContext(declaring, current); parent != nil {
			current = parent
			continue
		}
		if strings.TrimSpace(current.Superclass) == "" {
			return nativeObjectCloneCapability{}, false
		}
		origin := declaredJavaTypeOrigin(symbol.JavaType{
			Original:              current.Superclass,
			TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring)),
		}, declaring)
		if !completeJavaTypeOrigin(origin) || origin.nominalScope != nil || origin.parameter != nil || origin.arrayRank != 0 || origin.wildcard || len(origin.arguments) != 0 {
			return nativeObjectCloneCapability{}, false
		}
		contract, registered := nativeObjectCloneContracts[origin.externalName]
		if !registered || contract.canonicalOwner != origin.externalName || contract.superclass != "java.lang.Object" || contract.selectedOwner != "java.lang.Object" || contract.descriptor != "()Ljava/lang/Object;" || contract.methodAccess != 0x0104 || contract.declaresClone || contract.classSHA == "" || contract.selectedClassSHA == "" || contract.sourceSHA == "" || contract.selectedSourceSHA == "" || contract.stateCopier == "" {
			return nativeObjectCloneCapability{}, false
		}
		// Reuse the already-owned native storage projection; do not synthesize a
		// selector from a Java basename or from the physical Go receiver shape.
		if !canonicalReaderNativeEdge(current, declaring) {
			return nativeObjectCloneCapability{}, false
		}
		selector := superclassEmbeddedSelectorName(current, declaring)
		if selector == "" {
			return nativeObjectCloneCapability{}, false
		}
		return nativeObjectCloneCapability{root: current, contract: contract, storageSelector: selector}, true
	}
	return nativeObjectCloneCapability{}, false
}
