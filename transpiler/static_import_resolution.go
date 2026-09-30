package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type staticMethodImport struct {
	owner, member string
	wildcard      bool
}
type staticImportMethodResolution struct {
	source    *methodResolution
	intrinsic string
	problem   string
}

func staticMethodImports(ctx Ctx) []staticMethodImport {
	if ctx.currentFile == nil {
		return nil
	}
	// Imports belong to the file, including when hierarchy resolution changes
	// currentFile while retaining another class as its invocation context.
	// Pair this file's declaration tree with the source bytes read below.
	scope := ctx.currentFile.BaseClass
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil
	}
	root := scope.Class.DeclarationNode
	for root.Parent() != nil {
		root = root.Parent()
	}
	var result []staticMethodImport
	seen := map[staticMethodImport]bool{}
	for _, node := range nodeutil.NamedChildrenOf(root) {
		if node.Type() != "import_declaration" {
			continue
		}
		static, wildcard := false, false
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			static = static || child.Type() == "static"
			wildcard = wildcard || child.Content(ctx.currentFile.Source) == "*"
		}
		if !static || node.NamedChildCount() == 0 {
			continue
		}
		text := node.NamedChild(0).Content(ctx.currentFile.Source)
		entry := staticMethodImport{owner: text, wildcard: wildcard}
		if !wildcard {
			i := strings.LastIndex(text, ".")
			if i < 0 {
				continue
			}
			entry.owner, entry.member = text[:i], text[i+1:]
		}
		if !seen[entry] {
			seen[entry] = true
			result = append(result, entry)
		}
	}
	return result
}

// Method names declared in the lexical hierarchy hide imported names even when
// none of their overloads accepts the invocation arguments.
func lexicalMethodNamePresent(name string, ctx Ctx) bool {
	for lexical := ctx.currentClass; lexical != nil; lexical = lexical.Enclosing {
		seen := map[*symbol.ClassScope]bool{}
		var visit func(*symbol.ClassScope, bool) bool
		visit = func(scope *symbol.ClassScope, inherited bool) bool {
			if scope == nil || seen[scope] {
				return false
			}
			seen[scope] = true
			for _, def := range scope.Methods {
				if def == nil || def.Constructor || def.OriginalName != name {
					continue
				}
				if inherited {
					if def.IsPrivate || scope.IsInterface && def.IsStatic {
						continue
					}
					if !scope.IsInterface && !staticImportMethodHasModifier(def, "public") && !staticImportMethodHasModifier(def, "protected") && findJavaPackageForClassScope(scope) != findJavaPackageForClassScope(lexical) {
						continue
					}
				}
				return true
			}
			if visit(resolveSuperclassScopeInDeclaringContext(ctx, scope), true) {
				return true
			}
			for _, parent := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope) {
				if visit(parent, true) {
					return true
				}
			}
			return false
		}
		if visit(lexical, false) {
			return true
		}
	}
	return false
}

func staticImportMethodHasModifier(def *symbol.Definition, modifier string) bool {
	if def.DeclarationNode == nil {
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(def.DeclarationNode) {
		if child.Type() != "modifiers" {
			continue
		}
		for i := 0; i < int(child.ChildCount()); i++ {
			if child.Child(i).Type() == modifier {
				return true
			}
		}
	}
	return false
}

// Import applicability needs declared signatures; argument count cannot be
// reconstructed from a lowering callback or an inferred result type.
var staticIntrinsicImportSignatures = map[intrinsicKey][]*symbol.Definition{}

func registerStaticIntrinsicImportSignature(class, method string, parameters ...string) {
	def := &symbol.Definition{OriginalName: method, IsStatic: true}
	for _, parameter := range parameters {
		def.Parameters = append(def.Parameters, &symbol.Definition{OriginalType: parameter})
	}
	key := intrinsicKey{class, method}
	staticIntrinsicImportSignatures[key] = append(staticIntrinsicImportSignatures[key], def)
}

func staticImportHiddenSignature(def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	declaring := classScopeCtx(owner, ctx)
	declaring.localScope = def
	parts := []string{def.OriginalName}
	for index, parameter := range def.Parameters {
		typ := qualifyDeclaredReferenceType(symbol.JavaType{Original: genericMethodErasedJavaType(def, parameter.OriginalType)}, declaring)
		base, rank := javaArrayTypeParts(typ)
		base, _ = parseJavaTypeString(base)
		if executionParameterIsVariadic(def, index) {
			rank++
		}
		parts = append(parts, base+strings.Repeat("[]", rank))
	}
	return strings.Join(parts, "|")
}

func staticImportMethodAccessible(def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if def.IsPrivate {
		return false
	}
	if findJavaPackageForClassScope(owner) == findJavaPackageForClassScope(ctx.currentClass) {
		return true
	}
	if owner.IsInterface {
		return true
	}
	return staticImportMethodHasModifier(def, "public")
}

// resolveStaticImportedMethod retains declaration identity and uses the same
// applicability scores as source invocations. It does not fabricate a receiver
// AST or evaluate a class qualifier for an imported method.
func resolveStaticImportedMethod(invocation *sitter.Node, ctx Ctx, source []byte) staticImportMethodResolution {
	if invocation == nil || invocation.Type() != "method_invocation" || invocation.ChildByFieldName("object") != nil {
		return staticImportMethodResolution{}
	}
	nameNode := invocation.ChildByFieldName("name")
	if nameNode == nil {
		return staticImportMethodResolution{}
	}
	name := nameNode.Content(source)
	imports := staticMethodImports(ctx)
	if len(imports) == 0 {
		return staticImportMethodResolution{}
	}
	if lexicalMethodNamePresent(name, ctx) {
		return staticImportMethodResolution{}
	}
	var explicit, onDemand []staticMethodImport
	for _, entry := range imports {
		if entry.wildcard {
			onDemand = append(onDemand, entry)
		} else if entry.member == name {
			explicit = append(explicit, entry)
		}
	}
	entries := onDemand
	if len(explicit) > 0 {
		entries = explicit
	}
	var argNodes []*sitter.Node
	if args := invocation.ChildByFieldName("arguments"); args != nil {
		argNodes = nodeutil.NamedChildrenOf(args)
	}
	type candidate struct {
		resolution *methodResolution
		score      methodCandidateScore
		intrinsic  string
	}
	var candidates []candidate
	seen := map[*symbol.Definition]bool{}
	seenIntrinsics := map[string]bool{}
	unknownSignature := false
	matchedName := false
	for _, entry := range entries {
		scope := resolveClassScopeByQualifiedName(ctx, entry.owner)
		if scope == nil {
			owner, registered := canonicalIntrinsicOwner(entry.owner, ctx)
			if !registered || !intrinsicOwnerSupported(owner) {
				continue
			}
			key := intrinsicOwnerKey(owner)
			if _, exists := staticIntrinsics[intrinsicKey{key, name}]; exists {
				matchedName = true
				if !seenIntrinsics[key] {
					seenIntrinsics[key] = true
					signatures, known := staticIntrinsicImportSignatures[intrinsicKey{key, name}]
					if !known {
						unknownSignature = true
					}
					for _, def := range signatures {
						if score, ok := scoreMethodCandidate(def, nil, methodCandidateTypeParameterNames(nil, def), argNodes, ctx, source); ok {
							candidates = append(candidates, candidate{resolution: &methodResolution{def: def}, score: score, intrinsic: key})
						}
					}
				}
			}
			continue
		}
		hiddenSignatures := map[string]bool{}
		for current := scope; current != nil; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
			for _, def := range current.Methods {
				if def == nil || def.Constructor || !def.IsStatic || def.OriginalName != name || !staticImportMethodAccessible(def, current, ctx) {
					continue
				}
				signature := staticImportHiddenSignature(def, current, ctx)
				if hiddenSignatures[signature] {
					continue
				}
				hiddenSignatures[signature] = true
				matchedName = true
				if seen[def] {
					continue
				}
				seen[def] = true
				if !methodInvocationArityApplicable(def, len(argNodes)) {
					continue
				}
				score, ok := scoreMethodCandidate(def, current, methodCandidateTypeParameterNames(current, def), argNodes, ctx, source)
				if ok {
					candidates = append(candidates, candidate{resolution: &methodResolution{def: def, owner: current, receiverScope: scope, expandVarargsArray: score.expandVarargsArray}, score: score})
				}
			}
		}
	}
	if unknownSignature {
		return staticImportMethodResolution{problem: "unmodeled static import signature " + name}
	}

	if len(candidates) == 0 {
		if matchedName || len(explicit) > 0 {
			return staticImportMethodResolution{problem: "no applicable static method import " + name}
		}
		return staticImportMethodResolution{}
	}
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if methodCandidateScoreBetter(candidate.score, best.score) || (!methodCandidateScoreBetter(best.score, candidate.score) && methodResolutionMoreSpecific(candidate.resolution, best.resolution, ctx)) {
			best = candidate
		}
	}
	for _, candidate := range candidates {
		if candidate.resolution.def == best.resolution.def || methodCandidateScoreBetter(best.score, candidate.score) {
			continue
		}
		if !methodResolutionMoreSpecific(best.resolution, candidate.resolution, ctx) {
			return staticImportMethodResolution{problem: "ambiguous static method import " + name}
		}
	}
	if best.intrinsic != "" {
		return staticImportMethodResolution{intrinsic: best.intrinsic}
	}
	return staticImportMethodResolution{source: best.resolution}
}
