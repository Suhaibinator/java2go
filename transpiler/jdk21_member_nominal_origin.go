package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Member identity and completeness come from self InnerClasses and actual
// superclass/interface metadata. A package top-level certificate is never a
// certificate for a type's members. Binary names are opaque, including '$'.
type jdk21MemberDeclaration struct {
	binaryName, declaringOwner, simpleName          string
	public, static, isInterface, preview, synthetic bool
	sourceChain                                     []string
	signature, classSHA, sourceSHA                  string
}
type jdk21MemberOwner struct {
	certificate, superclass string
	interfaces              []string
	members                 map[string]string
}

func jdk21MemberEligible(d jdk21MemberDeclaration) bool {
	if d.binaryName == "" || !d.public || !d.static || d.preview || d.synthetic || d.classSHA == "" || d.sourceSHA == "" || len(d.sourceChain) < 2 {
		return false
	}
	top, presence := jdk21NominalQualified(d.sourceChain[0])
	if presence != jdk21NominalPresent || !jdk21NominalEligible(top) {
		return false
	}
	owner := d.declaringOwner
	seen := map[string]bool{d.binaryName: true}
	for {
		parent, member := jdk21MemberDeclarations[owner]
		if !member {
			return owner == top.binaryName
		}
		if seen[owner] || !parent.public || !parent.static || parent.preview || parent.synthetic || parent.classSHA == "" || parent.sourceSHA == "" {
			return false
		}
		seen[owner] = true
		owner = parent.declaringOwner
	}
}

// A direct declaration hides inherited declarations even when inaccessible.
// An incomplete ancestor makes absence/uniqueness unknown, never guessed.
func lookupJDK21Member(owner, name string) (jdk21MemberDeclaration, jdk21NominalPresence) {
	if name == "" || strings.ContainsAny(name, ".[/; ") {
		return jdk21MemberDeclaration{}, jdk21NominalUnknown
	}
	var visit func(string, map[string]bool) (jdk21MemberDeclaration, jdk21NominalPresence)
	visit = func(owner string, path map[string]bool) (jdk21MemberDeclaration, jdk21NominalPresence) {
		record, covered := jdk21MemberOwners[owner]
		if !covered || record.certificate == "" || path[owner] {
			return jdk21MemberDeclaration{}, jdk21NominalUnknown
		}
		if binary, present := record.members[name]; present {
			d, known := jdk21MemberDeclarations[binary]
			if !known || d.declaringOwner != owner || d.simpleName != name {
				return jdk21MemberDeclaration{}, jdk21NominalUnknown
			}
			return d, jdk21NominalPresent
		}
		next := make(map[string]bool, len(path)+1)
		for k, v := range path {
			next[k] = v
		}
		next[owner] = true
		var found jdk21MemberDeclaration
		for _, parent := range append(append([]string(nil), record.interfaces...), record.superclass) {
			if parent == "" {
				continue
			}
			d, p := visit(parent, next)
			if p == jdk21NominalUnknown {
				return jdk21MemberDeclaration{}, p
			}
			if p == jdk21NominalPresent {
				if found.binaryName != "" && found.binaryName != d.binaryName {
					return jdk21MemberDeclaration{}, jdk21NominalUnknown
				}
				found = d
			}
		}
		if found.binaryName != "" {
			return found, jdk21NominalPresent
		}
		return jdk21MemberDeclaration{}, jdk21NominalAbsentCertified
	}
	return visit(owner, map[string]bool{})
}

// Resolve source-style qualified native paths through certified top levels and
// their actual members, retaining the declaration owner across inherited aliases.
func jdk21QualifiedMemberPath(path string) (string, bool) {
	parts := strings.Split(path, ".")
	for i := 1; i <= len(parts); i++ {
		top, p := jdk21NominalQualified(strings.Join(parts[:i], "."))
		if p != jdk21NominalPresent {
			continue
		}
		if !jdk21NominalEligible(top) {
			return "", false
		}
		owner := top.binaryName
		for _, name := range parts[i:] {
			d, p := lookupJDK21Member(owner, name)
			if p != jdk21NominalPresent || !jdk21MemberEligible(d) {
				return "", false
			}
			owner = d.binaryName
		}
		return owner, true
	}
	return "", false
}

// Return a native top-level/member owner in this declaration's TYPE namespace.
// This helper does not admit APIs and does not use intrinsic simple-name tables.
func jdk21MemberUseOwner(base string, bindings map[string]*symbol.TypeParamDeclaration, ctx Ctx) (string, bool) {
	first := strings.SplitN(base, ".", 2)[0]
	if !jdk21NominalDeclaringUseOwned(ctx, first) || bindings[first] != nil || resolveClassScopeByQualifiedName(ctx, first) != nil {
		return "", false
	}
	if pkg, explicit := ctx.currentFile.Imports[first]; explicit {
		return jdk21QualifiedMemberPath(pkg + "." + base)
	}
	if strings.Contains(base, ".") {
		// Every demand namespace must certify that it cannot bind the prefix.
		if !jdk21MemberDemandPrefixAbsent(first, ctx) {
			return "", false
		}
		return jdk21QualifiedMemberPath(base)
	}
	candidate := ""
	add := func(owner string) bool {
		if candidate != "" && candidate != owner {
			return false
		}
		candidate = owner
		return true
	}
	if d, p := lookupJDK21TopLevel("java.lang", base); p == jdk21NominalPresent {
		if !jdk21NominalEligible(d) || !add(d.binaryName) {
			return "", false
		}
	}
	for _, pkg := range ordinaryTypeOnDemandImports(ctx) {
		if d, p := lookupJDK21TopLevel(pkg, base); p != jdk21NominalUnknown {
			if p == jdk21NominalPresent && (!jdk21NominalEligible(d) || !add(d.binaryName)) {
				return "", false
			}
			continue
		}
		if owner, known := jdk21QualifiedMemberPath(pkg); known {
			d, p := lookupJDK21Member(owner, base)
			if p == jdk21NominalUnknown || p == jdk21NominalPresent && (!jdk21MemberEligible(d) || !add(d.binaryName)) {
				return "", false
			}
			continue
		}
		// Source imports retain their namespace authority. Actual source candidates
		// have already won in resolveClassScopeByQualifiedName; unknown is held.
		if findQualifiedSourceClass(pkg) != nil || symbol.GlobalScope.FindPackage(pkg) != nil {
			continue
		}
		return "", false
	}
	for _, imported := range staticMethodImports(ctx) {
		if !imported.wildcard && imported.member != base {
			continue
		}
		owner, known := jdk21QualifiedMemberPath(imported.owner)
		if !known {
			return "", false
		}
		d, p := lookupJDK21Member(owner, base)
		if p == jdk21NominalUnknown || p == jdk21NominalPresent && (!jdk21MemberEligible(d) || !add(d.binaryName)) {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func jdk21MemberDemandPrefixAbsent(first string, ctx Ctx) bool {
	for _, pkg := range ordinaryTypeOnDemandImports(ctx) {
		if _, p := lookupJDK21TopLevel(pkg, first); p != jdk21NominalUnknown {
			if p != jdk21NominalAbsentCertified {
				return false
			}
			continue
		}
		if owner, known := jdk21QualifiedMemberPath(pkg); known {
			if _, p := lookupJDK21Member(owner, first); p != jdk21NominalAbsentCertified {
				return false
			}
			continue
		}
		if findQualifiedSourceClass(pkg) != nil || symbol.GlobalScope.FindPackage(pkg) != nil {
			continue
		}
		return false
	}
	for _, imported := range staticMethodImports(ctx) {
		if !imported.wildcard && imported.member != first {
			continue
		}
		owner, known := jdk21QualifiedMemberPath(imported.owner)
		if !known {
			return false
		}
		if _, p := lookupJDK21Member(owner, first); p != jdk21NominalAbsentCertified {
			return false
		}
	}
	return true
}

// A registered source ancestor's own type declaration hides native inherited
// members. Unknown parents and distinct inherited declarations cannot certify a
// unique member. Every edge switches to its real declaration file/header.
func jdk21SourceInheritedMember(scope *symbol.ClassScope, name string, ctx Ctx) (jdk21MemberDeclaration, jdk21NominalPresence) {
	var visit func(*symbol.ClassScope, map[*symbol.ClassScope]bool) (jdk21MemberDeclaration, jdk21NominalPresence)
	visit = func(owner *symbol.ClassScope, path map[*symbol.ClassScope]bool) (jdk21MemberDeclaration, jdk21NominalPresence) {
		if owner == nil || owner.Class == nil || path[owner] {
			return jdk21MemberDeclaration{}, jdk21NominalUnknown
		}
		file := findFileScopeForClassScope(owner, ctx)
		if file == nil {
			return jdk21MemberDeclaration{}, jdk21NominalUnknown
		}
		for _, child := range owner.Subclasses {
			if child.Class != nil && child.Class.OriginalName == name {
				return jdk21MemberDeclaration{}, jdk21NominalPresent
			}
		}
		next := make(map[*symbol.ClassScope]bool, len(path)+1)
		for k, v := range path {
			next[k] = v
		}
		next[owner] = true
		declaring := ctx.Clone()
		declaring.currentFile = nil
		declaring = classHeaderTypeCtx(owner, declaring)
		if declaring.currentFile != file {
			return jdk21MemberDeclaration{}, jdk21NominalUnknown
		}
		bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring))
		var found jdk21MemberDeclaration
		for _, parent := range append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass) {
			if strings.TrimSpace(parent) == "" {
				continue
			}
			base, _ := parseJavaTypeString(parent)
			var d jdk21MemberDeclaration
			var presence jdk21NominalPresence
			if source := resolveClassScopeByQualifiedName(declaring, base); source != nil {
				d, presence = visit(source, next)
			} else {
				native, known := jdk21MemberUseOwner(base, bindings, declaring)
				if !known {
					return jdk21MemberDeclaration{}, jdk21NominalUnknown
				}
				d, presence = lookupJDK21Member(native, name)
			}
			if presence == jdk21NominalUnknown {
				return jdk21MemberDeclaration{}, presence
			}
			if presence == jdk21NominalPresent {
				if !jdk21MemberEligible(d) || found.binaryName != "" && found.binaryName != d.binaryName {
					return jdk21MemberDeclaration{}, jdk21NominalUnknown
				}
				found = d
			}
		}
		if found.binaryName != "" {
			return found, jdk21NominalPresent
		}
		return jdk21MemberDeclaration{}, jdk21NominalAbsentCertified
	}
	return visit(scope, map[*symbol.ClassScope]bool{})
}

func declaredJDK21SourceMemberNominalOwner(base string, prefix *symbol.ClassScope, bindings map[string]*symbol.TypeParamDeclaration, ctx Ctx) (string, bool) {
	parts := strings.Split(base, ".")
	if len(parts) < 2 || !jdk21NominalDeclaringUseOwned(ctx, parts[0]) || bindings[parts[0]] != nil {
		return "", false
	}
	// The caller supplied an actual source prefix; consume precisely that prefix.
	var suffix []string
	for i := 1; i < len(parts); i++ {
		if resolveClassScopeByQualifiedName(ctx, strings.Join(parts[:i], ".")) == prefix {
			suffix = parts[i:]
			break
		}
	}
	if len(suffix) == 0 {
		return "", false
	}
	d, p := jdk21SourceInheritedMember(prefix, suffix[0], ctx)
	if p != jdk21NominalPresent || !jdk21MemberEligible(d) {
		return "", false
	}
	owner := d.binaryName
	for _, name := range suffix[1:] {
		d, p = lookupJDK21Member(owner, name)
		if p != jdk21NominalPresent || !jdk21MemberEligible(d) {
			return "", false
		}
		owner = d.binaryName
	}
	return owner, true
}

func declaredJDK21MemberNominalOwner(base string, bindings map[string]*symbol.TypeParamDeclaration, ctx Ctx) (string, bool) {
	first := strings.SplitN(base, ".", 2)[0]
	if !jdk21NominalDeclaringUseOwned(ctx, first) || bindings[first] != nil {
		return "", false
	}
	if !strings.Contains(base, ".") {
		// Lexical source members precede imported types. This walk preserves the
		// actual native declaring owner of a member inherited by a source class.
		for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
			d, p := jdk21SourceInheritedMember(scope, base, ctx)
			if p == jdk21NominalUnknown {
				return "", false
			}
			if p == jdk21NominalPresent {
				return d.binaryName, jdk21MemberEligible(d)
			}
		}
	} else {
		parts := strings.Split(base, ".")
		for i := 1; i < len(parts); i++ {
			if prefix := resolveClassScopeByQualifiedName(ctx, strings.Join(parts[:i], ".")); prefix != nil {
				return declaredJDK21SourceMemberNominalOwner(base, prefix, bindings, ctx)
			}
		}
	}
	owner, known := jdk21MemberUseOwner(base, bindings, ctx)
	if !known {
		return "", false
	}
	if d, member := jdk21MemberDeclarations[owner]; member {
		return owner, jdk21MemberEligible(d)
	}
	// A member-demand namespace can certify absence of a competing String (for
	// example); the old top-level-only resolver cannot certify that namespace.
	// Ordinary package-only resolutions continue through the unchanged old path.
	for _, namespace := range ordinaryTypeOnDemandImports(ctx) {
		if _, memberOwner := jdk21QualifiedMemberPath(namespace); memberOwner {
			return owner, true
		}
	}
	for _, imported := range staticMethodImports(ctx) {
		if _, memberOwner := jdk21QualifiedMemberPath(imported.owner); memberOwner {
			return owner, true
		}
	}
	return "", false
}
