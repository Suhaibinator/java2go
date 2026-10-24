package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

type declaredArgumentEquivalence uint8

const (
	declaredArgumentUnknown declaredArgumentEquivalence = iota
	declaredArgumentEquivalent
	declaredArgumentDifferent
)

type declaredBoundRelation uint8

const (
	declaredBoundUnknown declaredBoundRelation = iota
	declaredBoundYes
	declaredBoundNo
)

// The proof view retains wildcard syntax separately from the existing readable
// origin projection. Missing syntax is unknown, never a new nominal identity.
type declaredProofArgument struct {
	origin    inferredJavaTypeOrigin
	variance  string
	arguments []declaredProofArgument
}

func declaredProofOwnerHeader(origin inferredJavaTypeOrigin, ctx Ctx) bool {
	if origin.unresolved || origin.parameter != nil || origin.wildcard || origin.arrayRank != 0 || !completeJavaTypeOrigin(origin) {
		return false
	}
	if origin.nominalScope != nil {
		return declaredRefusalRegisteredScopeFile(origin.nominalScope, ctx) != nil && len(origin.nominalScope.OwnTypeParameters()) == len(origin.arguments)
	}
	record, ok := certifiedJDK21DeclarationBounds(origin.externalName)
	return ok && len(record.parameters) == len(origin.arguments)
}

func declaredInvariantArgumentEquivalence(actual, formal inferredJavaTypeOrigin, ctx Ctx) declaredArgumentEquivalence {
	a, ok := declaredArgumentProof(actual, actual.javaType, ctx, 0)
	if !ok {
		return declaredArgumentUnknown
	}
	f, ok := declaredArgumentProof(formal, formal.javaType, ctx, 0)
	if !ok {
		return declaredArgumentUnknown
	}
	return declaredCompareProofArguments(a, f, ctx, 0)
}

func declaredArgumentProof(origin inferredJavaTypeOrigin, written string, ctx Ctx, depth int) (declaredProofArgument, bool) {
	if depth > 64 || origin.parameter != nil || origin.unresolved || !completeJavaTypeOrigin(origin) || origin.arrayRank < 0 {
		return declaredProofArgument{}, false
	}
	written = strings.TrimSpace(written)
	view := declaredProofArgument{origin: origin}
	if origin.wildcard {
		if written == "?" {
			if origin.externalName != "java.lang.Object" || origin.nominalScope != nil || origin.arrayRank != 0 || len(origin.arguments) != 0 {
				return declaredProofArgument{}, false
			}
			view.variance = "unbounded"
			return view, true
		}
		if !strings.HasPrefix(written, "?") {
			return declaredProofArgument{}, false
		}
		tail := strings.TrimSpace(strings.TrimPrefix(written, "?"))
		if !strings.HasPrefix(tail, "extends") {
			return declaredProofArgument{}, false
		}
		tail = strings.TrimPrefix(tail, "extends")
		if len(tail) == 0 || !strings.ContainsRune(" \t\r\n\f", rune(tail[0])) {
			return declaredProofArgument{}, false
		}
		plain := origin
		plain.wildcard = false
		upper, ok := declaredArgumentProof(plain, strings.TrimSpace(tail), ctx, depth+1)
		if !ok {
			return declaredProofArgument{}, false
		}
		upper.variance = "extends"
		return upper, true
	}
	if written == "" || strings.HasPrefix(written, "?") {
		return declaredProofArgument{}, false
	}
	base, rank := javaArrayTypeParts(written)
	_, args := parseJavaTypeString(base)
	if rank != origin.arrayRank || len(args) != len(origin.arguments) {
		return declaredProofArgument{}, false
	}
	if origin.nominalScope != nil {
		if declaredRefusalRegisteredScopeFile(origin.nominalScope, ctx) == nil || len(origin.nominalScope.OwnTypeParameters()) != len(origin.arguments) {
			return declaredProofArgument{}, false
		}
	} else {
		record, ok := certifiedJDK21DeclarationBounds(origin.externalName)
		if !ok || len(record.parameters) != len(origin.arguments) {
			return declaredProofArgument{}, false
		}
	}
	for i := range args {
		arg, ok := declaredArgumentProof(origin.arguments[i], args[i], ctx, depth+1)
		if !ok {
			return declaredProofArgument{}, false
		}
		view.arguments = append(view.arguments, arg)
	}
	// Even when two proper heads differ, an unavailable wildcard declaration
	// must not become a negative proof through that earlier comparison.
	for i, arg := range view.arguments {
		if arg.variance != "" {
			if _, ok := declaredGenericFormalBounds(view, i, ctx, depth+1); !ok {
				return declaredProofArgument{}, false
			}
		}
	}
	return view, true
}

func declaredCompareProofArguments(actual, formal declaredProofArgument, ctx Ctx, depth int) declaredArgumentEquivalence {
	if depth > 64 || actual.variance != "" || formal.variance != "" {
		return declaredArgumentUnknown
	}
	a, f := actual.origin, formal.origin
	if a.parameter != nil || f.parameter != nil || a.unresolved || f.unresolved {
		return declaredArgumentUnknown
	}
	if a.nominalScope != f.nominalScope || a.externalName != f.externalName || a.arrayRank != f.arrayRank {
		return declaredArgumentDifferent
	}
	if len(actual.arguments) != len(formal.arguments) {
		return declaredArgumentUnknown
	}
	result := declaredArgumentEquivalent
	for i, aa := range actual.arguments {
		ff := formal.arguments[i]
		var eq declaredArgumentEquivalence
		if aa.variance != "" || ff.variance != "" {
			if aa.variance == "" || ff.variance == "" {
				return declaredArgumentUnknown
			}
			eq = declaredCompareWildcards(actual, formal, i, ctx, depth+1)
		} else {
			eq = declaredCompareProofArguments(aa, ff, ctx, depth+1)
		}
		if eq == declaredArgumentDifferent {
			return eq
		}
		if eq == declaredArgumentUnknown {
			result = eq
		}
	}
	return result
}

func declaredCompareWildcards(actual, formal declaredProofArgument, slot int, ctx Ctx, depth int) declaredArgumentEquivalence {
	a, ok := declaredEffectiveWildcardUpper(actual, slot, ctx, depth+1)
	if !ok {
		return declaredArgumentUnknown
	}
	f, ok := declaredEffectiveWildcardUpper(formal, slot, ctx, depth+1)
	if !ok {
		return declaredArgumentUnknown
	}
	// Identical ordered intersections are a proof; different or unnormalized
	// intersections are deliberately outside this bounded comparison.
	if len(a) == len(f) {
		equal := true
		for i := range a {
			equal = equal && declaredCompareProofArguments(a[i], f[i], ctx, depth+1) == declaredArgumentEquivalent
		}
		if equal {
			return declaredArgumentEquivalent
		}
	}
	if len(a) != 1 || len(f) != 1 {
		return declaredArgumentUnknown
	}
	ab := declaredProofBoundSubtype(a[0], f[0], ctx, depth+1)
	ba := declaredProofBoundSubtype(f[0], a[0], ctx, depth+1)
	if ab == declaredBoundYes && ba == declaredBoundYes {
		return declaredArgumentEquivalent
	}
	if ab == declaredBoundYes && ba == declaredBoundNo || ba == declaredBoundYes && ab == declaredBoundNo {
		return declaredArgumentDifferent
	}
	return declaredArgumentUnknown
}

func declaredEffectiveWildcardUpper(owner declaredProofArgument, slot int, ctx Ctx, depth int) ([]declaredProofArgument, bool) {
	if depth > 64 || slot < 0 || slot >= len(owner.arguments) {
		return nil, false
	}
	arg := owner.arguments[slot]
	if arg.variance != "unbounded" && arg.variance != "extends" {
		return nil, false
	}
	bounds, ok := declaredGenericFormalBounds(owner, slot, ctx, depth+1)
	if !ok {
		return nil, false
	}
	if arg.variance == "extends" {
		arg.variance = ""
		bounds = append(bounds, arg)
	}
	// Remove only a conjunct proved redundant; never drop an unproved edge.
	for i := 0; i < len(bounds); i++ {
		for j := i + 1; j < len(bounds); {
			if declaredProofBoundSubtype(bounds[i], bounds[j], ctx, depth+1) == declaredBoundYes {
				bounds = append(bounds[:j], bounds[j+1:]...)
			} else if declaredProofBoundSubtype(bounds[j], bounds[i], ctx, depth+1) == declaredBoundYes {
				bounds[i] = bounds[j]
				bounds = append(bounds[:j], bounds[j+1:]...)
				i = -1
				break
			} else {
				j++
			}
		}
	}
	return bounds, len(bounds) > 0
}

func declaredGenericFormalBounds(owner declaredProofArgument, slot int, ctx Ctx, depth int) ([]declaredProofArgument, bool) {
	if depth > 64 || slot < 0 || slot >= len(owner.arguments) || owner.variance != "" || owner.origin.arrayRank != 0 {
		return nil, false
	}
	bindings := map[*symbol.TypeParamDeclaration]declaredProofArgument{}
	var bounds []declaredProofArgument
	if scope := owner.origin.nominalScope; scope != nil {
		file := declaredRefusalRegisteredScopeFile(scope, ctx)
		params := scope.OwnTypeParameters()
		if file == nil || len(params) != len(owner.arguments) || slot >= len(params) {
			return nil, false
		}
		for i, p := range params {
			if p.Declaration == nil {
				return nil, false
			}
			if _, duplicate := bindings[p.Declaration]; duplicate {
				return nil, false
			}
			bindings[p.Declaration] = owner.arguments[i]
		}
		declaring := ctx.Clone()
		declaring.currentFile = file
		declaring.currentClass = scope
		declaring.localScope = nil
		for _, b := range params[slot].Bounds {
			origin := declaredJavaTypeOrigin(b, declaring)
			if replacement, exists := bindings[origin.parameter]; origin.parameter != nil && exists {
				if replacement.variance != "" {
					return nil, false
				}
				bounds = append(bounds, replacement)
				continue
			}
			origin = substituteJavaTypeOrigin(origin, declaredProofOriginBindings(bindings))
			view, ok := declaredArgumentProof(origin, b.Original, declaring, depth+1)
			if !ok {
				return nil, false
			}
			bounds = append(bounds, view)
		}
	} else {
		record, ok := certifiedJDK21DeclarationBounds(owner.origin.externalName)
		if !ok || len(record.parameters) != len(owner.arguments) || slot >= len(record.parameters) {
			return nil, false
		}
		for i, p := range record.parameters {
			bindings[p.declaration] = owner.arguments[i]
		}
		p := record.parameters[slot]
		if !p.known || p.classBoundAbsent != (p.classBound == nil) {
			return nil, false
		}
		recipes := append([]*declaredBoundRecipe{}, p.interfaceBounds...)
		if p.classBound != nil {
			recipes = append([]*declaredBoundRecipe{p.classBound}, recipes...)
		}
		for _, recipe := range recipes {
			view, ok := declaredBoundRecipeProof(recipe, bindings, ctx, map[*declaredBoundRecipe]bool{}, depth+1)
			if !ok {
				return nil, false
			}
			bounds = append(bounds, view)
		}
	}
	if len(bounds) == 0 {
		object := inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: "java.lang.Object"}
		view, ok := declaredArgumentProof(object, object.javaType, ctx, depth+1)
		if !ok {
			return nil, false
		}
		bounds = append(bounds, view)
	}
	return bounds, true
}

func declaredProofOriginBindings(values map[*symbol.TypeParamDeclaration]declaredProofArgument) map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin {
	result := map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
	for p, v := range values {
		result[p] = v.origin
	}
	return result
}

func declaredBoundRecipeProof(recipe *declaredBoundRecipe, bindings map[*symbol.TypeParamDeclaration]declaredProofArgument, ctx Ctx, active map[*declaredBoundRecipe]bool, depth int) (declaredProofArgument, bool) {
	if !validDeclaredBoundRecipeShape(recipe, map[*declaredBoundRecipe]bool{}, depth) {
		return declaredProofArgument{}, false
	}
	return walkDeclaredBoundRecipeProof(recipe, bindings, ctx, active, depth)
}

// Unsupported proof recipes remain intact. Validate their complete grammar so
// absent variance, missing components and cycles cannot masquerade as facts.
func validDeclaredBoundRecipeShape(recipe *declaredBoundRecipe, active map[*declaredBoundRecipe]bool, depth int) bool {
	if recipe == nil || depth > 64 || active[recipe] {
		return false
	}
	active[recipe] = true
	defer delete(active, recipe)
	switch recipe.kind {
	case "nominal":
		if recipe.externalName == "" || recipe.parameter != nil || recipe.variance != "" || recipe.bound != nil || recipe.arrayRank != 0 {
			return false
		}
		for _, arg := range recipe.arguments {
			if !validDeclaredBoundRecipeShape(arg, active, depth+1) {
				return false
			}
		}
		return true
	case "parameter":
		return recipe.parameter != nil && recipe.externalName == "" && recipe.variance == "" && recipe.bound == nil && recipe.arrayRank == 0 && len(recipe.arguments) == 0
	case "wildcard":
		if recipe.parameter != nil || recipe.externalName != "" || recipe.arrayRank != 0 || len(recipe.arguments) != 0 {
			return false
		}
		if recipe.variance == "unbounded" {
			return recipe.bound == nil
		}
		return (recipe.variance == "extends" || recipe.variance == "super") && validDeclaredBoundRecipeShape(recipe.bound, active, depth+1)
	case "array":
		return recipe.parameter == nil && recipe.externalName == "" && recipe.variance == "" && len(recipe.arguments) == 0 && recipe.arrayRank > 0 && recipe.arrayRank <= 255 && validDeclaredBoundRecipeShape(recipe.bound, active, depth+1)
	}
	return false
}

func walkDeclaredBoundRecipeProof(recipe *declaredBoundRecipe, bindings map[*symbol.TypeParamDeclaration]declaredProofArgument, ctx Ctx, active map[*declaredBoundRecipe]bool, depth int) (declaredProofArgument, bool) {
	if recipe == nil || depth > 64 || active[recipe] {
		return declaredProofArgument{}, false
	}
	active[recipe] = true
	defer delete(active, recipe)
	if recipe.kind == "parameter" {
		value, ok := bindings[recipe.parameter]
		return value, ok && value.variance == ""
	}
	if recipe.kind != "nominal" {
		return declaredProofArgument{}, false
	}
	record, ok := certifiedJDK21DeclarationBounds(recipe.externalName)
	if !ok || len(record.parameters) != len(recipe.arguments) {
		return declaredProofArgument{}, false
	}
	view := declaredProofArgument{origin: inferredJavaTypeOrigin{externalName: recipe.externalName, javaType: recipe.externalName}}
	for _, arg := range recipe.arguments {
		child, ok := walkDeclaredBoundRecipeProof(arg, bindings, ctx, active, depth+1)
		if !ok {
			return declaredProofArgument{}, false
		}
		view.arguments = append(view.arguments, child)
		view.origin.arguments = append(view.origin.arguments, child.origin)
	}
	return view, true
}

func declaredProofBoundSubtype(actual, formal declaredProofArgument, ctx Ctx, depth int) declaredBoundRelation {
	if depth > 64 || actual.variance != "" || formal.variance != "" {
		return declaredBoundUnknown
	}
	if declaredCompareProofArguments(actual, formal, ctx, depth+1) == declaredArgumentEquivalent {
		return declaredBoundYes
	}
	if actual.origin.arrayRank != 0 || formal.origin.arrayRank != 0 {
		return declaredBoundUnknown
	}
	if actual.origin.nominalScope != nil {
		if formal.origin.nominalScope == nil && formal.origin.externalName == "java.lang.Object" && len(formal.arguments) == 0 && declaredRefusalRegisteredScopeFile(actual.origin.nominalScope, ctx) != nil {
			return declaredBoundYes
		}
		return declaredBoundUnknown
	}
	if formal.origin.nominalScope != nil {
		return declaredBoundUnknown
	}
	return jdk21DeclaredBoundSubtype(actual.origin, formal.origin)
}

// No is distinct from unavailable: the entire actual raw superclass/interface
// closure must validate before absence of a target path can prove anything.
func jdk21DeclaredBoundSubtype(actual, formal inferredJavaTypeOrigin) declaredBoundRelation {
	if actual.parameter != nil || formal.parameter != nil || actual.unresolved || formal.unresolved || actual.wildcard || formal.wildcard || actual.arrayRank != 0 || formal.arrayRank != 0 || actual.nominalScope != nil || formal.nominalScope != nil {
		return declaredBoundUnknown
	}
	a, ok := certifiedJDK21DeclarationBounds(actual.externalName)
	if !ok || len(a.parameters) != len(actual.arguments) {
		return declaredBoundUnknown
	}
	f, ok := certifiedJDK21DeclarationBounds(formal.externalName)
	if !ok || len(f.parameters) != len(formal.arguments) {
		return declaredBoundUnknown
	}
	relation := jdk21DeclaredBoundSubtypeWithRecords(actual.externalName, formal.externalName, jdk21MemberOwners, jdk21MemberOwners)
	// A closed absence of the raw declaration path also excludes every typed
	// instantiation. A present path cannot establish typed argument conversion.
	if relation == declaredBoundNo {
		return relation
	}
	if len(f.parameters) != 0 {
		return declaredBoundUnknown
	}
	return relation
}

func jdk21DeclaredBoundSubtypeWithRecords(actual, formal string, records, attested map[string]jdk21MemberOwner) declaredBoundRelation {
	target, ok := records[formal]
	targetProof, targetKnown := attested[formal]
	if !ok || !targetKnown || target.certificate == "" || target.certificate != targetProof.certificate || target.superclass != targetProof.superclass || len(target.interfaces) != len(targetProof.interfaces) {
		return declaredBoundUnknown
	}
	for i := range target.interfaces {
		if target.interfaces[i] != targetProof.interfaces[i] {
			return declaredBoundUnknown
		}
	}
	states := map[string]uint8{}
	found := false
	var visit func(string) bool
	visit = func(owner string) bool {
		if states[owner] == 1 {
			return false
		}
		if states[owner] == 2 {
			return true
		}
		r, ok := records[owner]
		p, proof := attested[owner]
		if !ok || !proof || r.certificate == "" || r.certificate != p.certificate || r.superclass != p.superclass || len(r.interfaces) != len(p.interfaces) {
			return false
		}
		for i := range r.interfaces {
			if r.interfaces[i] != p.interfaces[i] {
				return false
			}
		}
		states[owner] = 1
		found = found || owner == formal
		if r.superclass != "" && !visit(r.superclass) {
			return false
		}
		for _, parent := range r.interfaces {
			if !visit(parent) {
				return false
			}
		}
		states[owner] = 2
		return true
	}
	if !visit(actual) {
		return declaredBoundUnknown
	}
	if found {
		return declaredBoundYes
	}
	return declaredBoundNo
}
