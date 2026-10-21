package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// External identity is established in the declaration's TYPE namespace. A
// fully qualified spelling does not bypass a source type or captured binder.
func declaredExternalNominalOwner(base string, bindings map[string]*symbol.TypeParamDeclaration, ctx Ctx) string {
	first := strings.SplitN(base, ".", 2)[0]
	if bindings[first] != nil {
		return ""
	}
	if prefix := resolveClassScopeByQualifiedName(ctx, first); prefix != nil {
		if owner, proved := declaredJDK21SourceMemberNominalOwner(base, prefix, bindings, ctx); proved {
			return owner
		}
		return ""
	}
	if owner, proved := declaredJDK21MemberNominalOwner(base, bindings, ctx); proved {
		return owner
	}
	known := func(owner string) bool {
		if _, native := sqlDateRuntimeNames[owner]; native {
			return ctx.currentFile != nil && ctx.currentClass != nil && findFileScopeForClassScope(ctx.currentClass, ctx) == ctx.currentFile
		}
		if strings.HasPrefix(owner, "java.lang.reflect.") {
			name := strings.TrimPrefix(owner, "java.lang.reflect.")
			declaration, present := declaredJavaLangReflectTypes[name]
			_, supported := reflectProtocolConstants[name]
			return present && declaration.public && declaration.isInterface && supported
		}
		if knownDeclaredExternalInterface(owner) {
			return true
		}
		if strings.HasPrefix(owner, "java.util.") && !strings.Contains(strings.TrimPrefix(owner, "java.util."), ".") {
			v, ok := declaredJavaUtilTypes[strings.TrimPrefix(owner, "java.util.")]
			return ok && v.public
		}
		switch owner {
		case "java.nio.ByteBuffer", "java.nio.channels.FileChannel", "java.io.RandomAccessFile":
			// A registered operation is not a declaration-origin certificate.
			// These native type identities require their actual declaring file.
			visible := visibleTypeParameterDeclarationForJavaType(first, ctx)
			return (visible == nil || visible.SourceName != first) && ctx.currentFile != nil && ctx.currentClass != nil && findFileScopeForClassScope(ctx.currentClass, ctx) == ctx.currentFile
		case "java.lang.StringBuilder", "java.lang.StringBuffer", "java.io.StringReader":
			// These new native declarations require the actual source use's owner.
			// A stale or missing donor file is not declaration evidence.
			visible := visibleTypeParameterDeclarationForJavaType(first, ctx)
			if visible != nil && visible.SourceName == first || ctx.currentFile == nil || ctx.currentClass == nil || findFileScopeForClassScope(ctx.currentClass, ctx) != ctx.currentFile {
				return false
			}
			return true
		case "java.lang.Object", "java.lang.String", "java.lang.Class", "java.lang.Number", "java.lang.Boolean", "java.lang.Byte", "java.lang.Short", "java.lang.Character", "java.lang.Integer", "java.lang.Long", "java.lang.Float", "java.lang.Double", "java.lang.Void", "java.lang.Enum", "java.lang.Throwable", "java.lang.Exception", "java.lang.RuntimeException", "java.lang.Error":
			return true
		}
		declaration, presence := jdk21NominalQualified(owner)
		return presence == jdk21NominalPresent && jdk21NominalEligible(declaration) && jdk21NominalDeclaringUseOwned(ctx, first)
	}
	if strings.Contains(base, ".") {
		if ctx.currentFile != nil {
			if _, ok := ctx.currentFile.Imports[first]; ok {
				return ""
			}
		}
		if !declaredInterfaceDemandPrefixAbsent(first, ctx) {
			return ""
		}
		if known(base) {
			return base
		}
		return ""
	}
	if ctx.currentFile != nil {
		if pkg, ok := ctx.currentFile.Imports[base]; ok {
			owner := pkg + "." + base
			if known(owner) {
				return owner
			}
			return ""
		}
	}
	owner := ""
	if known("java.lang." + base) {
		owner = "java.lang." + base
	}
	for _, pkg := range ordinaryTypeOnDemandImports(ctx) {
		if pkg == "java.lang" {
			continue
		}
		if pkg == "java.util" {
			v, exists := declaredJavaUtilTypes[base]
			if !exists || !v.public {
				continue
			}
			candidate := pkg + "." + base
			if owner != "" && owner != candidate {
				return ""
			}
			owner = candidate
			continue
		}
		if pkg == "java.lang.reflect" {
			declaration, present := declaredJavaLangReflectTypes[base]
			if !present || !declaration.public {
				continue
			}
			candidate := pkg + "." + base
			if !known(candidate) || owner != "" && owner != candidate {
				return ""
			}
			owner = candidate
			continue
		}
		if pkg == "java.sql" {
			declaration, present := declaredJavaSQLTypes[base]
			if !present || !declaration.public {
				continue
			}
			candidate := pkg + "." + base
			if !known(candidate) || owner != "" && owner != candidate {
				return ""
			}
			owner = candidate
			continue
		}
		// A finite presence proof does not infer any other package member's
		// absence. Noncandidates retain the existing source-package fallback.
		switch candidate := pkg + "." + base; candidate {
		case "java.nio.ByteBuffer", "java.nio.channels.FileChannel", "java.io.RandomAccessFile":
			if !known(candidate) || owner != "" && owner != candidate {
				return ""
			}
			owner = candidate
			continue
		}
		declaration, presence := lookupJDK21TopLevel(pkg, base)
		if presence == jdk21NominalAbsentCertified {
			continue
		}
		if presence == jdk21NominalPresent {
			candidate := pkg + "." + base
			if !jdk21NominalEligible(declaration) || !known(candidate) || owner != "" && owner != candidate {
				return ""
			}
			owner = candidate
			continue
		}
		if symbol.GlobalScope.FindPackage(pkg) != nil || findQualifiedSourceClass(pkg) != nil {
			continue
		}
		return ""
	}
	return owner
}

func completeJavaTypeOrigin(value inferredJavaTypeOrigin) bool {
	if value.unresolved {
		return false
	}
	if value.nominalScope != nil || value.parameter != nil || value.externalName != "" {
		return true
	}
	base, _ := javaArrayTypeParts(value.javaType)
	_, primitive := javaPrimitiveType(base)
	return primitive || base == "void" || base == "null" || base == ternaryNullJavaType
}

func intrinsicJavaTypeOrigin(owner string) inferredJavaTypeOrigin {
	return inferredJavaTypeOrigin{externalName: owner, javaType: owner}
}

func methodFormalOrigins(def *symbol.Definition, owner *symbol.ClassScope, bindings map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin, ctx Ctx) []inferredJavaTypeOrigin {
	declaring, _ := invocationMethodDeclarationContext(def, ctx)
	if owner != nil {
		declaring = classScopeCtx(owner, ctx)
		declaring.localScope = def
		declaring.syntheticTypeParameters = nil
	}
	result := make([]inferredJavaTypeOrigin, len(def.Parameters))
	for i, p := range def.Parameters {
		if p != nil {
			result[i] = substituteJavaTypeOrigin(declaredJavaTypeOrigin(definitionDeclaredJavaType(p), declaring), bindings)
		}
	}
	return result
}

func ownerOriginsFromSpellings(owner *symbol.ClassScope, args []string, ctx Ctx) map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin {
	result := map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
	if owner == nil {
		return result
	}
	for i, p := range owner.TypeParameters {
		if i < len(args) {
			result[p.Declaration] = declaredJavaTypeOrigin(symbol.JavaType{Original: args[i], TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
		}
	}
	return result
}

// handled is authoritative for complete references, including rejection. Poly
// and unresolved method inference variables retain the established deferred path.
func declaredOriginConversion(actual, formal inferredJavaTypeOrigin, ctx Ctx) (cost int, exact, applicable, handled bool) {
	if actual.unresolved || formal.unresolved {
		return 0, false, false, true
	}
	if !completeJavaTypeOrigin(actual) || !completeJavaTypeOrigin(formal) {
		return 0, false, false, false
	}
	if actual.parameter != nil || formal.parameter != nil {
		if sameJavaTypeOrigin(actual, formal) {
			return 0, true, true, true
		}
		return 0, false, false, false
	}
	null := actual.javaType == "null" || actual.javaType == ternaryNullJavaType
	if null {
		_, primitive := javaPrimitiveType(formal.javaType)
		return 1, false, !primitive || formal.arrayRank > 0, true
	}
	ap := actual.nominalScope == nil && actual.externalName == ""
	fp := formal.nominalScope == nil && formal.externalName == ""
	if actual.arrayRank == 0 && formal.arrayRank == 0 && (ap || fp) {
		if ap && fp {
			return 0, false, false, false
		} // Existing primitive widening costs.
		// Loose boxing/unboxing may only target a proved java.lang wrapper.
		if ap {
			primitive, _ := javaPrimitiveType(actual.javaType)
			wrapper := ternaryBoxedJavaType(primitive)
			if formal.externalName == "java.lang."+wrapper && wrapper != "" {
				return 2, false, true, true
			}
			if formal.externalName == "java.lang.Object" || formal.externalName == "java.lang.Number" && primitive != "boolean" && primitive != "char" {
				return 3, false, true, true
			}
			return 0, false, false, true
		}
		primitive, ok := ternaryBoxedPrimitive(formal.javaType)
		_ = primitive
		_ = ok
		if actual.externalName != "" {
			unboxed, wrapper := ternaryBoxedPrimitive(actual.externalName)
			if wrapper {
				wanted, isPrimitive := javaPrimitiveType(formal.javaType)
				if isPrimitive {
					if unboxed == wanted {
						return 8, false, true, true
					}
					if distance, ok := javaPrimitiveWideningDistance(unboxed, wanted); ok {
						return 8 + distance, false, true, true
					}
				}
			}
		}
		return 0, false, false, true
	}
	if sameJavaTypeOrigin(actual, formal) {
		return 0, true, true, true
	}
	if formal.arrayRank == 0 && formal.externalName == "java.lang.Object" {
		return 2, false, true, true
	}
	if actual.arrayRank > 0 && formal.arrayRank == 0 && (formal.externalName == "java.lang.Cloneable" || formal.externalName == "java.io.Serializable") {
		return 2, false, true, true
	}
	if actual.arrayRank > 0 && formal.arrayRank > 0 {
		a, f := actual, formal
		a.arrayRank--
		f.arrayRank--
		_, actualPrimitive := javaPrimitiveType(stripOriginArrayDisplay(a.javaType))
		_, formalPrimitive := javaPrimitiveType(stripOriginArrayDisplay(f.javaType))
		if a.arrayRank == 0 && actualPrimitive || f.arrayRank == 0 && formalPrimitive {
			// An immediate primitive component is invariant; array covariance
			// cannot perform the scalar boxing conversion admitted below.
			equal := sameJavaTypeOrigin(a, f)
			return 0, equal, equal, true
		}
		// int[] and String[] are reference components of int[][] and
		// String[][]. They can widen to Object/Cloneable/Serializable even
		// when the complete source and target arrays have different ranks.
		return declaredOriginConversion(a, f, ctx)
	}
	if actual.arrayRank != formal.arrayRank {
		return 0, false, false, true
	}
	if actual.nominalScope != nil && formal.nominalScope != nil {
		if actual.nominalScope == formal.nominalScope && len(formal.arguments) > 0 {
			if len(actual.arguments) != len(formal.arguments) {
				return 0, false, false, false
			}
			for index, arg := range formal.arguments {
				if arg.parameter != nil || !completeJavaTypeOrigin(arg) {
					return 0, false, false, false
				}
				if arg.wildcard {
					continue
				}
				if !sameJavaTypeOrigin(actual.arguments[index], arg) {
					return 0, false, false, true
				}
			}
			return 0, true, true, true
		}
		return 1, false, javaReferenceTypeAssignable(actual.nominalScope, formal.nominalScope, ctx), true
	}
	if actual.nominalScope != nil && formal.externalName != "" {
		if !closedDeclaredAncestryOrigin(actual) || !closedDeclaredAncestryOrigin(formal) {
			return 0, false, false, false
		}
		return 1, false, declaredSourceExternalOriginAssignable(actual, formal, ctx), true
	}
	if actual.externalName != "" && formal.externalName != "" {
		if actual.externalName == formal.externalName {
			if len(actual.arguments) == 0 || len(formal.arguments) == 0 {
				return 0, true, true, true // Existing raw reference conversion.
			}
			if !closedDeclaredAncestryOrigin(actual) || !closedDeclaredAncestryOrigin(formal) {
				return 0, false, false, false
			}
			if len(actual.arguments) != len(formal.arguments) {
				return 0, false, false, true
			}
			for index, expected := range formal.arguments {
				if !sameJavaTypeOrigin(actual.arguments[index], expected) {
					return 0, false, false, true
				}
			}
			return 0, true, true, true
		}
		if canonicalThrowableOriginAssignable(actual, formal) {
			return 1, false, true, true
		}
		if threadSourceCanonicalOriginAssignable(actual, formal) {
			return 1, false, true, true
		}
		if actual.arrayRank == 0 && formal.arrayRank == 0 && !actual.wildcard && !formal.wildcard && len(actual.arguments) == 0 && len(formal.arguments) == 0 && formal.externalName == "java.lang.CharSequence" && (actual.externalName == "java.lang.StringBuilder" || actual.externalName == "java.lang.StringBuffer") {
			return 1, false, true, true
		}
		if formal.externalName == "java.lang.Number" {
			if primitive, wrapper := ternaryBoxedPrimitive(actual.externalName); wrapper && primitive != "boolean" && primitive != "char" {
				return 1, false, true, true
			}
		}
		if (actual.externalName == "java.lang.String" && (formal.externalName == "java.lang.CharSequence" || formal.externalName == "java.io.Serializable" || formal.externalName == "java.lang.Comparable")) || (actual.externalName == "java.lang.Class" && formal.externalName == "java.io.Serializable") {
			return 1, false, true, true
		}
	}
	return 0, false, false, true
}

// Preserve the previous deferred path for wildcard projections and live
// inference variables; closed declaration proofs never reenter text lookup.
func closedDeclaredAncestryOrigin(origin inferredJavaTypeOrigin) bool {
	if !completeJavaTypeOrigin(origin) || origin.parameter != nil || origin.wildcard {
		return false
	}
	for _, argument := range origin.arguments {
		if !closedDeclaredAncestryOrigin(argument) {
			return false
		}
	}
	return true
}

// Follow the same declaration-owned source ancestry as the runtime ancestry
// proof, retaining type identities across substitution. A canonical endpoint or
// supplied argument must never be parsed again in the invocation's namespace.
func declaredSourceExternalOriginAssignable(actual, formal inferredJavaTypeOrigin, ctx Ctx) bool {
	type ancestryView struct {
		origin inferredJavaTypeOrigin
		path   map[*symbol.ClassScope]bool
	}
	queue := []ancestryView{{origin: actual}}
	var seen []inferredJavaTypeOrigin
	for len(queue) > 0 {
		view := queue[0]
		queue = queue[1:]
		current := view.origin.nominalScope
		if current == nil || view.path[current] {
			continue
		}
		duplicate := false
		for _, previous := range seen {
			if sameJavaTypeOrigin(previous, view.origin) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		seen = append(seen, view.origin)
		path := make(map[*symbol.ClassScope]bool, len(view.path)+1)
		for scope := range view.path {
			path[scope] = true
		}
		path[current] = true
		declaring := classHeaderTypeCtx(current, ctx)
		declaring.syntheticTypeParameters = nil
		bindings := map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
		for index, parameter := range current.TypeParameters {
			if index < len(view.origin.arguments) {
				bindings[parameter.Declaration] = view.origin.arguments[index]
			}
		}
		raw := len(current.TypeParameters) > 0 && len(view.origin.arguments) == 0
		parents := append([]string{current.Superclass}, current.ImplementedInterfaces...)
		for _, parent := range parents {
			if strings.TrimSpace(parent) == "" {
				continue
			}
			edge := declaredJavaTypeOrigin(symbol.JavaType{
				Original:              parent,
				TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring)),
			}, declaring)
			edge = substituteJavaTypeOrigin(edge, bindings)
			if raw {
				// A raw source owner's supertypes are erased, rather than
				// populated with binders from the invocation's lexical scope.
				edge.arguments = nil
			}
			if edge.nominalScope != nil {
				queue = append(queue, ancestryView{origin: edge, path: path})
				continue
			}
			if edge.unresolved || edge.externalName == "" {
				continue
			}
			if declaredExternalEndpointAssignable(edge, formal, ctx) {
				return true
			}
		}
	}
	return false
}

func declaredExternalEndpointAssignable(actual, formal inferredJavaTypeOrigin, ctx Ctx) bool {
	// Both owners have already been proved in their own declarations. The
	// collection relation consumes only those canonical owner names, with no
	// type-argument text that a caller context could reinterpret.
	if actual.externalName != formal.externalName {
		_, _, applies, handled := declaredOriginConversion(actual, formal, ctx)
		ownerAssignable := handled && applies || builtinCollectionReferenceAssignable(actual.externalName, formal.externalName, nil)
		if !ownerAssignable {
			// These names are retained canonical endpoints, not names to resolve
			// in the caller. The parent service validates each canonical owner.
			seen := map[string]bool{}
			for owner := actual.externalName; owner != "" && !seen[owner]; {
				seen[owner] = true
				parent, known := builtinThrowableReferenceParent(owner)
				if !known {
					break
				}
				if parent == formal.externalName {
					ownerAssignable = true
					break
				}
				owner = parent
			}
		}
		if !ownerAssignable {
			return false
		}
	}
	if len(actual.arguments) == 0 || len(formal.arguments) == 0 {
		return true // Raw reference conversion.
	}
	if len(actual.arguments) != len(formal.arguments) {
		return false
	}
	for index, expected := range formal.arguments {
		if !sameJavaTypeOrigin(actual.arguments[index], expected) {
			return false
		}
	}
	return true
}

func scoreDeclaredMethodCandidate(def *symbol.Definition, owner *symbol.ClassScope, candidateTypeParams []string, argNodes []*sitter.Node, ctx Ctx, source []byte, ownerArgs []string, bindings map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin) (methodCandidateScore, bool) {
	if def == nil || !methodInvocationArityApplicable(def, len(argNodes)) {
		return methodCandidateScore{}, false
	}
	formals := methodFormalOrigins(def, owner, bindings, ctx)
	count := len(formals)
	variadic := count > 0 && executionParameterIsVariadic(def, count-1)
	fixed := false
	if variadic && len(argNodes) == count {
		inference := ctx.Clone()
		inference.expectedType = ""
		inference.expectedTypeRoot = nil
		var actual inferredJavaTypeOrigin
		typ, known := inferExprJavaType(argNodes[count-1], inference, source, &actual)
		actual.javaType = typ
		formal := formals[count-1]
		formal.arrayRank++
		_, _, applies, handled := declaredOriginConversion(actual, formal, ctx)
		if handled {
			fixed = applies
		} else {
			fixed = invocationArgumentCanTargetVarargsArray(argNodes[count-1], def.Parameters[count-1], def, owner, candidateTypeParams, ctx, source, ownerArgs)
		}
		_ = known
	}
	score := methodCandidateScore{expandVarargsArray: fixed}
	if variadic && !fixed {
		score.phase = 2
	}
	for i, node := range argNodes {
		index := i
		if variadic && index >= count-1 {
			index = count - 1
		}
		if index < 0 || index >= count {
			return methodCandidateScore{}, false
		}
		inference := ctx.Clone()
		inference.expectedType = ""
		inference.expectedTypeRoot = nil
		var actual inferredJavaTypeOrigin
		typ, _ := inferExprJavaType(node, inference, source, &actual)
		actual.javaType = typ
		formal := formals[index]
		if fixed && i == count-1 {
			formal.arrayRank++
		}
		cost, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
		if handled && cost >= 2 && actual.arrayRank == 0 && formal.arrayRank == 0 {
			_, primitive := javaPrimitiveType(typ)
			if primitive && formal.externalName != "" && score.phase < 1 {
				score.phase = 1
			}
		}
		if !handled {
			expected := receiverInstantiatedMethodFormal(def.Parameters[index], def, owner, ownerArgs, ctx)
			if fixed && i == count-1 {
				expected += "[]"
			}
			cost, exact, applies = javaInvocationConversionCost(node, expected, candidateTypeParams, ctx, source)
			if !applies {
				cost, exact, applies = javaLooseInvocationConversionCost(node, expected, candidateTypeParams, ctx, source)
				if applies && score.phase < 1 {
					score.phase = 1
				}
			}
			if applies {
				if projected, proved := declaredMethodArgumentInferredOrigin(formal, def, formals, argNodes, fixed, ctx, source); proved {
					if projectedCost, projectedExact, projectedApplies, projectedHandled := declaredOriginConversion(actual, projected, ctx); projectedHandled && projectedApplies && projectedExact {
						cost, exact = projectedCost, projectedExact
					}
				}
			}
		}
		if !applies {
			return methodCandidateScore{}, false
		}
		score.totalCost += cost
		if exact {
			score.exactCount++
		}
	}
	if !methodInvocationBoundsApplicable(def, owner, argNodes, fixed, ctx, source) {
		return methodCandidateScore{}, false
	}
	return score, true
}

func declaredOriginDescriptor(origin inferredJavaTypeOrigin, ctx Ctx, seen map[*symbol.TypeParamDeclaration]bool) (ast.Expr, bool) {
	if origin.unresolved {
		return nil, false
	}
	rank := origin.arrayRank
	origin.arrayRank = 0
	var value ast.Expr
	if origin.parameter != nil {
		if seen[origin.parameter] {
			return nil, false
		}
		seen[origin.parameter] = true
		defer delete(seen, origin.parameter)
		found := false
		for _, p := range visibleTypeParameterDeclarations(ctx) {
			if p.Declaration == origin.parameter {
				found = true
				if len(p.Bounds) > 0 {
					var ok bool
					value, ok = declaredOriginDescriptor(declaredJavaTypeOrigin(p.Bounds[0], ctx), ctx, seen)
					if !ok {
						return nil, false
					}
				} else {
					value = stdjavaQualifiedExpr("ObjectTypeID", ctx)
				}
				break
			}
		}
		if !found {
			return nil, false
		}
	} else if origin.nominalScope != nil {
		value = javaTypeIDLiteral(sourceClassRuntimeTypeID(origin.nominalScope, ctx), ctx)
	} else if origin.externalName != "" {
		if origin.externalName == "java.lang.Object" {
			value = stdjavaQualifiedExpr("ObjectTypeID", ctx)
		} else {
			value = javaTypeIDLiteral(origin.externalName, ctx)
		}
	} else {
		base, _ := javaArrayTypeParts(origin.javaType)
		if _, ok := javaPrimitiveType(base); ok || base == "void" {
			value = stdjavaCall(ctx, "PrimitiveTypeID", metadataString(base))
		} else {
			return nil, false
		}
	}
	for i := 0; i < rank; i++ {
		value = stdjavaCall(ctx, "ArrayTypeID", value)
	}
	return value, true
}

func canonicalResultPhysicalExpr(origin inferredJavaTypeOrigin, ctx Ctx) (ast.Expr, bool) {
	if !completeJavaTypeOrigin(origin) || origin.arrayRank != 0 || origin.parameter != nil || origin.nominalScope != nil {
		return nil, false
	}
	switch origin.externalName {
	case "java.lang.Object":
		return ast.NewIdent("any"), true
	case "java.lang.String":
		return javaStringReferenceType(ctx), true
	case "java.lang.Class":
		return &ast.StarExpr{X: stdjavaQualifiedExpr("Class", ctx)}, true
	}
	return nil, false
}

type canonicalBranchPlan struct {
	origin   inferredJavaTypeOrigin
	arms     []inferredJavaTypeOrigin
	physical ast.Expr
}

func canonicalBranchEligible(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	switch node.Type() {
	case "null_literal", "string_literal", "class_literal", "identifier", "field_access", "array_access":
		return true
	case "object_creation_expression":
		return objectCreationClassBody(node) == nil && !strings.Contains(node.ChildByFieldName("type").Content(source), "<>")
	case "cast_expression":
		return true
	case "binary_expression":
		return node.ChildByFieldName("operator").Content(source) == "+"
	case "ternary_expression":
		_, a, b := ternaryExpressionParts(node)
		return canonicalBranchEligible(a, ctx, source) && canonicalBranchEligible(b, ctx, source)
	}
	return false
}

func canonicalStandaloneBranchPlan(nodes []*sitter.Node, ctx Ctx, source []byte, allowObject bool) (*canonicalBranchPlan, bool) {
	if ctx.rawFunctionTarget.active(ctx) || ctx.genericInterfaceMethodBody != nil {
		return nil, false
	}
	if _, ok := currentErasedCallableOwnerTypeParameterErasure(ctx.expectedType, ctx); ok {
		return nil, false
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	var values []inferredJavaTypeOrigin
	for _, node := range nodes {
		if !canonicalBranchEligible(node, ctx, source) {
			return nil, false
		}
		var origin inferredJavaTypeOrigin
		typ, known := inferExprJavaType(node, inference, source, &origin)
		origin.javaType = typ
		if !known || !completeJavaTypeOrigin(origin) || origin.parameter != nil {
			return nil, false
		}
		values = append(values, origin)
	}
	nonnull := []inferredJavaTypeOrigin{}
	for _, value := range values {
		if value.javaType != "null" && value.javaType != ternaryNullJavaType {
			nonnull = append(nonnull, value)
		}
	}
	if len(nonnull) == 0 {
		return nil, false
	}
	result, known := joinedJavaTypeOrigin("", nonnull, ctx)
	if !known {
		return nil, false
	}
	if result.externalName == "java.lang.Object" {
		canonicalArm := false
		for _, value := range nonnull {
			if value.arrayRank == 0 && (value.externalName == "java.lang.Object" || value.externalName == "java.lang.String" || value.externalName == "java.lang.Class") {
				canonicalArm = true
			}
		}
		if !canonicalArm {
			return nil, false
		}
	}
	physical, ok := canonicalResultPhysicalExpr(result, ctx)
	if !ok || !allowObject && result.externalName == "java.lang.Object" {
		return nil, false
	}
	for _, v := range nonnull {
		_, _, applicable, handled := declaredOriginConversion(v, result, ctx)
		if !handled || !applicable {
			return nil, false
		}
	}
	return &canonicalBranchPlan{origin: result, arms: values, physical: physical}, true
}

func parseCanonicalPlanArm(node *sitter.Node, origin inferredJavaTypeOrigin, plan *canonicalBranchPlan, source []byte, ctx Ctx) ast.Expr {
	if origin.javaType == "null" || origin.javaType == ternaryNullJavaType {
		return ast.NewIdent("nil")
	}
	neutral := ctx.Clone()
	neutral.expectedType = ""
	neutral.expectedTypeRoot = nil
	value := ParseExpr(node, source, neutral)
	if plan.origin.externalName != "java.lang.Object" {
		return value
	}
	operand := ast.NewIdent("value")
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{operand}, Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.IfStmt{Cond: stdjavaCall(ctx, "JavaReferenceEqual", operand, ast.NewIdent("nil")), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}}, &ast.ReturnStmt{Results: []ast.Expr{operand}}}}}, Args: []ast.Expr{value}}
}

func canonicalSwitchArmNodes(body *sitter.Node) ([]*sitter.Node, bool) {
	var nodes []*sitter.Node
	for _, rule := range nodeutil.NamedChildrenOf(body) {
		if rule.Type() != "switch_rule" {
			return nil, false
		}
		found := false
		for _, child := range nodeutil.NamedChildrenOf(rule) {
			if child.Type() == "expression_statement" {
				nodes = append(nodes, child.NamedChild(0))
				found = true
			} else if child.Type() != "switch_label" {
				return nil, false
			}
		}
		if !found {
			return nil, false
		}
	}
	return nodes, len(nodes) > 0
}

func stripOriginArrayDisplay(value string) string { base, _ := javaArrayTypeParts(value); return base }

// Hidden member slots belong to the enclosing class declaration, whereas the
// written member arguments belong to its own parameters. Never align them by
// spelling or let the method's shadowing T supply the enclosing T slot.
func declarationMemberTypeOrigins(origin inferredJavaTypeOrigin, base string, javaType symbol.JavaType, ctx Ctx) inferredJavaTypeOrigin {
	scope := origin.nominalScope
	if scope == nil || scope.Enclosing == nil {
		return origin
	}
	hidden := len(scope.TypeParameters) - len(scope.OwnTypeParameters())
	if hidden == 0 || len(origin.arguments) != len(scope.OwnTypeParameters()) {
		return origin
	}
	receiver := ctx.currentClass
	arguments := classParameterTypeOrigins(receiver)
	// A written outer qualifier has its own original TYPE argument provenance.
	depth := 0
	separator := -1
	for index, c := range base {
		switch c {
		case '<':
			depth++
		case '>':
			depth--
		case '.':
			if depth == 0 {
				separator = index
			}
		}
	}
	if separator >= 0 {
		prefix := base[:separator]
		outer := declaredJavaTypeOrigin(symbol.JavaType{Original: prefix, TypeParameterBindings: javaType.TypeParameterBindings}, ctx)
		if outer.nominalScope != nil {
			if len(outer.arguments) == 0 && len(outer.nominalScope.TypeParameters) > 0 {
				// Explicitly raw outer syntax (including an imported raw member
				// qualifier) does not acquire the lexical class's generic slots.
				return origin
			}
			if len(outer.arguments) != len(outer.nominalScope.TypeParameters) {
				origin.unresolved = true
				return origin
			}
			receiver = outer.nominalScope
			arguments = outer.arguments
		}
	}
	// A complete lexical capture is already registered on the receiver. Resolve
	// it before ancestry edges that can themselves need these enclosing slots.
	lexical := declarationCapturedClassTypeOrigins(scope, receiver, arguments, ctx)
	if len(lexical) == hidden {
		captured := make([]inferredJavaTypeOrigin, 0, hidden+len(origin.arguments))
		for index := 0; index < hidden; index++ {
			value, known := lexical[scope.TypeParameters[index].Declaration]
			if !known || !completeJavaTypeOrigin(value) || value.wildcard {
				captured = nil
				break
			}
			captured = append(captured, value)
		}
		if len(captured) == hidden {
			origin.arguments = append(captured, origin.arguments...)
			return origin
		}
	}
	bindings := sourceClassOriginBindings(receiver, arguments, scope.Enclosing, ctx)
	captured := make([]inferredJavaTypeOrigin, 0, hidden+len(origin.arguments))
	for index := 0; index < hidden; index++ {
		declaration := scope.TypeParameters[index].Declaration
		value, known := bindings[declaration]
		if !known {
			value, known = declarationCapturedClassTypeOrigins(scope, receiver, arguments, ctx)[declaration]
		}
		if !known {
			value, known = localCapturedMethodTypeOrigin(scope, declaration, ctx)
		}
		if !known {
			origin.unresolved = true
			return origin
		}
		captured = append(captured, value)
	}
	origin.arguments = append(captured, origin.arguments...)
	return origin
}
