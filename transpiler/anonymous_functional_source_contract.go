package transpiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// This private certificate belongs to an actual allocation and one emission.
// Input snapshots certify bytes only. All declaration/role proofs below are
// repeated independently; neither a map entry nor an emission stamp is authority.
type anonymousFunctionalCertificate struct {
	self                                               *anonymousFunctionalCertificate
	graph                                              *symbol.GlobalSymbols
	file                                               *symbol.FileScope
	root, allocation, typ, body, ownerNode, executable *sitter.Node
	lexical                                            Ctx
	source                                             []byte
	sourceIdentity                                     *byte
	family, header                                     string
	arguments                                          []inferredJavaTypeOrigin
	info                                               *anonymousClassInfo
	scope                                              *symbol.ClassScope
	class                                              *symbol.Definition
	selected                                           *symbol.Definition
	phase                                              uint8
	sealed                                             bool
	shape, lexicalBindings                             string
	hoistStart, hoistEnd                               int
	emitted                                            []ast.Decl
	emittedShape                                       []string
}

const (
	anonymousFunctionalBody uint8 = iota + 1
	anonymousFunctionalComplete
	anonymousFunctionalFailed
)

func bytesEqualAnonymousFunctionalSource(a, b []byte) bool { return bytes.Equal(a, b) }
func anonymousFunctionalEqual(a, b *sitter.Node) bool      { return a != nil && b != nil && a.Equal(b) }
func anonymousFunctionalWithin(node, root *sitter.Node) bool {
	for current := node; current != nil; current = current.Parent() {
		if anonymousFunctionalEqual(current, root) {
			return true
		}
	}
	return false
}
func anonymousFunctionalActualChild(parent, child *sitter.Node) bool {
	if parent == nil || child == nil {
		return false
	}
	for i := 0; i < int(parent.ChildCount()); i++ {
		if anonymousFunctionalEqual(parent.Child(i), child) {
			return true
		}
	}
	return false
}
func anonymousFunctionalRegistered(file *symbol.FileScope, owner *symbol.ClassScope) bool {
	if file == nil || owner == nil {
		return false
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files {
			if f != file {
				continue
			}
			for _, top := range f.TopLevelClasses {
				if classScopeContains(top, owner) {
					return true
				}
			}
		}
	}
	return false
}

// Original info/scope/certificate membership is tested BEFORE mutable node kind.
// An unregistered synthetic scope also cannot become a named Function by losing
// its map entry or borrowing a registered named declaration node.
func anonymousFunctionalLookup(scope *symbol.ClassScope, ctx Ctx) (*anonymousFunctionalCertificate, bool) {
	if scope == nil {
		return nil, false
	}
	// This check precedes every mutable map, node-kind and registration route.
	// Missing/replaced cache information can only refuse a marked source scope.
	if scope.HasAnonymousSourceOrigin() {
		return anonymousFunctionalOwned(scope, ctx), true
	}
	if certificate := anonymousFunctionalOwned(scope, ctx); certificate != nil {
		return certificate, true
	}
	for _, info := range ctx.anonymousClasses {
		if info != nil && (info.scope == scope || info.functional != nil && info.functional.scope == scope) {
			return nil, true
		}
	}
	if scope.Class != nil && scope.Class.DeclarationNode != nil && scope.Class.DeclarationNode.Type() == "object_creation_expression" {
		return nil, true
	}
	if anonymousFunctionalActualNamed(scope, ctx) {
		return nil, false
	}
	return nil, true
}

func anonymousFunctionalOwned(scope *symbol.ClassScope, ctx Ctx) *anonymousFunctionalCertificate {
	var found *anonymousFunctionalCertificate
	for key, info := range ctx.anonymousClasses {
		if info == nil {
			continue
		}
		c := info.functional
		if info.scope != scope && (c == nil || c.scope != scope) {
			continue
		}
		if c == nil || c.self != c || c.info != info || c.scope != scope || info.scope != scope {
			return nil
		}
		original, ok := anonymousClassSourceKey(c.allocation)
		if !ok || original != key || found != nil {
			return nil
		}
		found = c
	}
	return found
}

// Neither unmarkedness nor membership in localClasses/TYPE maps proves a named
// declaration. Check the actual declaration role and registered containing AST.
// Named family predicates below the dispatcher remain unchanged.
func anonymousFunctionalActualNamed(scope *symbol.ClassScope, ctx Ctx) bool {
	return anonymousFunctionalActualNamedSeen(scope, ctx, map[*symbol.ClassScope]bool{})
}
func anonymousFunctionalActualNamedSeen(scope *symbol.ClassScope, ctx Ctx, active map[*symbol.ClassScope]bool) bool {
	if active[scope] {
		return false
	}
	active[scope] = true
	defer delete(active, scope)
	if scope == nil || scope.HasAnonymousSourceOrigin() || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return false
	}
	node := scope.Class.DeclarationNode
	switch node.Type() {
	case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
	default:
		return false
	}
	var file *symbol.FileScope
	registered := false
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, candidate := range pkg.Files {
			if candidate == nil {
				continue
			}
			for _, top := range candidate.TopLevelClasses {
				if classScopeContains(top, scope) {
					if file != nil && file != candidate {
						return false
					}
					file = candidate
					registered = true
				}
			}
		}
	}
	if !registered {
		associated := false
		for name, info := range ctx.localClasses {
			if info != nil && info.scope == scope && info.structName == scope.Class.Name && node.ChildByFieldName("name") != nil && name == scope.Class.OriginalName {
				associated = true
			}
		}
		if !associated {
			return false
		}
		// A genuinely local named declaration is inside an executable of its actual
		// lexical owner. A donated registered top-level Donor cannot have that role.
		owner := scope.Enclosing
		if owner == nil || owner == scope {
			return false
		}
		var anchor, body *sitter.Node
		var owned bool
		file, anchor, body, owned = anonymousLexicalOwnerFrame(owner, ctx, active)
		if !owned {
			return false
		}
		nearestOwner, executable := anonymousLexicalNearestOwner(node)
		if executable == nil || !anonymousFunctionalEqual(nearestOwner, anchor) || !anonymousLexicalDeclaredMethod(owner, body, executable, file.Source) {
			return false
		}
	}
	if file == nil || !anonymousFunctionalNamespace(file) || !anonymousFunctionalWithin(node, file.ParsedRoot()) || node.ChildByFieldName("name") == nil || node.ChildByFieldName("name").Content(file.Source) != scope.Class.OriginalName {
		return false
	}
	var written []string
	if superclass := node.ChildByFieldName("superclass"); superclass != nil {
		types := supplierHeaderTypeNodes(superclass)
		if len(types) != 1 {
			return false
		}
		written = append(written, types[0].Content(file.Source))
	} else {
		written = append(written, "")
	}
	for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
		if child.Type() == "super_interfaces" || child.Type() == "interfaces" || child.Type() == "extends_interfaces" {
			for _, typ := range supplierHeaderTypeNodes(child) {
				written = append(written, typ.Content(file.Source))
			}
		}
	}
	metadata := append([]string{scope.Superclass}, scope.ImplementedInterfaces...)
	if len(written) != len(metadata) {
		return false
	}
	for i, text := range written {
		if text != metadata[i] {
			return false
		}
	}
	return true
}

func anonymousFunctionalBindings(params []symbol.TypeParam) string {
	var b strings.Builder
	for _, p := range params {
		fmt.Fprintf(&b, "%q:%p:%q:%q|", p.Name, p.Declaration, p.EmittedName(), func() string {
			if p.Declaration == nil {
				return ""
			}
			return p.Declaration.SourceName
		}())
		for _, bound := range p.Bounds {
			fmt.Fprintf(&b, "%q:%s;", bound.Original, anonymousReceiverTypeBindings(bound.TypeParameterBindings))
		}
	}
	return b.String()
}
func anonymousFunctionalDefinition(d *symbol.Definition) string {
	if d == nil {
		return "nil"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%p:%p:%q:%q:%q:%q:%p:%s:%s:%t:%t:%t:%t:%t:%t:%t:%t:%t:%t:%q|", d, d.DeclarationNode, d.OriginalName, d.Name, d.OriginalType, d.Type, d.DirectTypeParameter, anonymousReceiverTypeBindings(d.TypeParameterBindings), anonymousFunctionalBindings(d.TypeParameters), d.Nullable, d.IsStatic, d.IsVariadic, d.IsFinal, d.IsPrivate, d.HasBody, d.RuntimeDefault, d.RequiresHelper, d.Constructor, d.IsCompileTimeConstant, d.HelperName)
	for _, p := range d.Parameters {
		b.WriteString(anonymousFunctionalDefinition(p))
	}
	return b.String()
}
func anonymousFunctionalAST(n ast.Node) string {
	if n == nil {
		return "nil"
	}
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), n); err != nil {
		return "invalid"
	}
	return b.String()
}
func anonymousFunctionalShape(info *anonymousClassInfo) string {
	if info == nil || info.scope == nil || info.scope.Class == nil {
		return "invalid"
	}
	s := info.scope
	var b strings.Builder
	fmt.Fprintf(&b, "%p:%p:%p:%p:%q:%q:%q:%q:%q:%t:%t:%t:%t:%t:%t:%s:%s|", info, s, s.Class, s.Enclosing, info.structName, info.fieldInitializerMethodName, s.Superclass, s.ImplementedInterfaces, s.EnclosingField, s.IsInterface, s.IsAbstract, s.IsEnum, s.IsInner, s.HasInstanceFieldInitializers, s.DeclaredTypeParameters != nil, anonymousFunctionalBindings(s.TypeParameters), anonymousFunctionalBindings(s.DeclaredTypeParameters))
	b.WriteString(anonymousFunctionalDefinition(s.Class))
	for _, d := range s.Fields {
		b.WriteString(anonymousFunctionalDefinition(d))
	}
	b.WriteByte('|')
	for _, d := range info.declaredFields {
		b.WriteString(anonymousFunctionalDefinition(d))
	}
	b.WriteByte('|')
	for _, d := range s.Methods {
		b.WriteString(anonymousFunctionalDefinition(d))
	}
	for _, cap := range info.captured {
		fmt.Fprintf(&b, "%q:%q:%s:%s|", cap.name, cap.fieldName, anonymousFunctionalAST(cap.goType), anonymousFunctionalDefinition(cap.javaDef))
	}
	for _, nested := range s.Subclasses {
		fmt.Fprintf(&b, "%p|", nested)
	}
	return b.String()
}

// Check each source binder against its actual containing declaration/order.
// Emitted names are retained ABI, independently fingerprinted for freshness.
func anonymousFunctionalDeclaredBinders(node *sitter.Node, params []symbol.TypeParam, source []byte) bool {
	var names []string
	if n := node.ChildByFieldName("type_parameters"); n != nil {
		for _, p := range nodeutil.NamedChildrenOf(n) {
			if p.Type() == "type_parameter" && p.NamedChild(0) != nil {
				names = append(names, p.NamedChild(0).Content(source))
			}
		}
	}
	if len(names) != len(params) {
		return false
	}
	fresh := symbol.ExtractTypeParameters(node.ChildByFieldName("type_parameters"), source)
	for i, p := range params {
		if len(p.Bounds) != len(fresh[i].Bounds) {
			return false
		}
		for j, b := range p.Bounds {
			if b.Original != fresh[i].Bounds[j].Original {
				return false
			}
		}
	}
	for i, n := range names {
		p := params[i]
		if p.Declaration == nil || p.Name != n || p.Declaration.SourceName != n || p.EmittedName() == "" {
			return false
		}
	}
	return true
}
func anonymousFunctionalLexicalProof(c *anonymousFunctionalCertificate) bool {
	ctx := c.lexical
	if !anonymousFunctionalNamespace(c.file) || !anonymousFunctionalExecutable(c) || c.self != c || c.graph != symbol.GlobalScope || ctx.currentFile != c.file || c.file == nil || c.file.SourceInputSnapshot() == nil || c.file.SourceInputSnapshot().Validate() != nil || !bytes.Equal(c.file.Source, c.source) || anonymousReceiverSourceIdentity(c.file.Source) != c.sourceIdentity || !anonymousFunctionalEqual(c.file.ParsedRoot(), c.root) {
		return false
	}
	owner := ctx.currentClass
	if owner == nil || owner.Class == nil || !anonymousFunctionalRegistered(c.file, owner) || !anonymousFunctionalEqual(owner.Class.DeclarationNode, c.ownerNode) || !anonymousFunctionalWithin(c.allocation, c.root) || !anonymousFunctionalWithin(c.ownerNode, c.root) {
		return false
	}
	if !anonymousFunctionalActualChild(c.allocation, c.typ) || !anonymousFunctionalEqual(c.allocation.ChildByFieldName("type"), c.typ) || !anonymousFunctionalActualChild(c.allocation, c.body) || !anonymousFunctionalEqual(objectCreationClassBody(c.allocation), c.body) {
		return false
	}
	var nearestOwner, executable *sitter.Node
	for p := c.allocation.Parent(); p != nil; p = p.Parent() {
		if executable == nil && (p.Type() == "method_declaration" || p.Type() == "constructor_declaration") {
			executable = p
		}
		if p.Type() == "class_declaration" || p.Type() == "enum_declaration" || p.Type() == "record_declaration" || p.Type() == "interface_declaration" {
			nearestOwner = p
			break
		}
	}
	if c.ownerNode.ChildByFieldName("name") == nil || c.ownerNode.ChildByFieldName("name").Content(c.source) != owner.Class.OriginalName {
		return false
	}
	if !anonymousFunctionalEqual(nearestOwner, c.ownerNode) {
		return false
	}
	if c.executable == nil {
		if executable != nil || ctx.localScope != nil {
			return false
		}
	} else {
		if !anonymousFunctionalEqual(executable, c.executable) || ctx.localScope == nil || !anonymousFunctionalEqual(ctx.localScope.DeclarationNode, c.executable) {
			return false
		}
		found := false
		for _, m := range owner.Methods {
			found = found || m == ctx.localScope
		}
		if !found || !anonymousFunctionalDeclaredBinders(c.executable, ctx.localScope.TypeParameters, c.source) {
			return false
		}
	}
	for current := owner; current != nil; current = current.Enclosing {
		if current.Class == nil || !anonymousFunctionalRegistered(c.file, current) || !anonymousFunctionalDeclaredBinders(current.Class.DeclarationNode, current.OwnTypeParameters(), c.source) {
			return false
		}
	}
	return true
}

func anonymousFunctionalOrigin(t string, ctx Ctx) inferredJavaTypeOrigin {
	return declaredJavaTypeOrigin(symbol.JavaType{Original: t, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
}
func anonymousFunctionalOriginOwned(o inferredJavaTypeOrigin, ctx Ctx) bool {
	if !completeJavaTypeOrigin(o) || o.wildcard {
		return false
	}
	if o.parameter != nil {
		found := false
		for _, p := range visibleTypeParameterDeclarations(ctx) {
			found = found || p.Declaration == o.parameter
		}
		if !found {
			return false
		}
	}
	if o.nominalScope != nil {
		f := findFileScopeForClassScope(o.nominalScope, ctx)
		if !anonymousFunctionalRegistered(f, o.nominalScope) || o.nominalScope.Class == nil || o.nominalScope.Class.DeclarationNode == nil || f.SourceInputSnapshot() == nil || f.SourceInputSnapshot().Validate() != nil || !anonymousFunctionalWithin(o.nominalScope.Class.DeclarationNode, f.ParsedRoot()) {
			return false
		}
	}
	for _, a := range o.arguments {
		if !anonymousFunctionalOriginOwned(a, ctx) {
			return false
		}
	}
	return true
}

// Diamond uses a real parent declaration, never ctx.expectedType display text.
func anonymousFunctionalTarget(node *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	value := node
	for p := value.Parent(); p != nil && p.Type() == "parenthesized_expression"; p = value.Parent() {
		value = p
	}
	p := value.Parent()
	if p == nil {
		return inferredJavaTypeOrigin{}, false
	}
	if p.Type() == "return_statement" && anonymousFunctionalActualChild(p, value) && ctx.localScope != nil && ctx.localScope.DeclarationNode != nil {
		typ := ctx.localScope.DeclarationNode.ChildByFieldName("type")
		if typ != nil && typ.Content(source) == ctx.localScope.OriginalType {
			return anonymousFunctionalOrigin(typ.Content(source), ctx), true
		}
	}
	if p.Type() == "assignment_expression" && anonymousFunctionalEqual(p.ChildByFieldName("right"), value) {
		left := p.ChildByFieldName("left")
		if left != nil && left.Type() == "identifier" && ctx.currentClass != nil {
			name := left.Content(source)
			// A local declaration wins over the class field, regardless of display.
			if ctx.localScope != nil && ctx.localScope.FindVariable(name) != nil {
				return inferredJavaTypeOrigin{}, false
			}
			var field *symbol.Definition
			for _, candidate := range ctx.currentClass.Fields {
				if candidate != nil && candidate.OriginalName == name {
					if field != nil {
						return inferredJavaTypeOrigin{}, false
					}
					field = candidate
				}
			}
			if field != nil && field.DeclarationNode != nil && field.DeclarationNode.Type() == "field_declaration" && ctx.currentClass.Class != nil && anonymousFunctionalActualChild(ctx.currentClass.Class.DeclarationNode.ChildByFieldName("body"), field.DeclarationNode) {
				typ := field.DeclarationNode.ChildByFieldName("type")
				actual := false
				for _, declarator := range syntheticFieldDeclarators(field.DeclarationNode) {
					if n := declarator.ChildByFieldName("name"); n != nil && n.Content(source) == name {
						actual = true
					}
				}
				if actual && typ != nil && typ.Content(source) == field.OriginalType {
					declaring := classHeaderTypeCtx(ctx.currentClass, ctx)
					return declaredJavaTypeOrigin(definitionDeclaredJavaType(field), declaring), true
				}
			}
		}
	}
	if p.Type() == "variable_declarator" && anonymousFunctionalEqual(p.ChildByFieldName("value"), value) {
		declaration := p.Parent()
		if declaration != nil && (declaration.Type() == "field_declaration" || declaration.Type() == "local_variable_declaration") {
			if typ := declaration.ChildByFieldName("type"); typ != nil {
				return anonymousFunctionalOrigin(typ.Content(source), ctx), true
			}
		}
	}
	return inferredJavaTypeOrigin{}, false
}
func anonymousFunctionalHeader(c *anonymousFunctionalCertificate) (inferredJavaTypeOrigin, bool) {
	t := c.typ.Content(c.source)
	base, _ := parseJavaTypeString(t)
	if !anonymousFunctionalPrefixOwned(t, c.allocation, c.lexical, c.source) || nativeFunctionalFamily(base, c.lexical) != c.family {
		return inferredJavaTypeOrigin{}, false
	}
	origin := anonymousFunctionalOrigin(t, c.lexical)
	if strings.Contains(t, "<>") {
		target, ok := anonymousFunctionalTarget(c.allocation, c.lexical, c.source)
		if !ok || target.externalName != "java.util.function."+c.family || target.nominalScope != nil || target.parameter != nil || target.arrayRank != 0 {
			return origin, false
		}
		origin = target
	} else if len(origin.arguments) == 0 {
		count := 1
		if c.family == "Function" {
			count = 2
		}
		for i := 0; i < count; i++ {
			origin.arguments = append(origin.arguments, intrinsicJavaTypeOrigin("java.lang.Object"))
		}
	}
	count := 1
	if c.family == "Function" {
		count = 2
	}
	if origin.externalName != "java.util.function."+c.family || origin.nominalScope != nil || origin.parameter != nil || origin.arrayRank != 0 || len(origin.arguments) != count {
		return origin, false
	}
	for _, a := range origin.arguments {
		if !anonymousFunctionalOriginOwned(a, c.lexical) {
			return origin, false
		}
	}
	return origin, true
}
func anonymousFunctionalStart(node, typ, body *sitter.Node, source []byte, ctx Ctx) (*anonymousFunctionalCertificate, bool) {
	if node == nil || typ == nil || body == nil || node.Type() != "object_creation_expression" || typ.EndByte() > uint32(len(source)) || body.EndByte() > uint32(len(source)) || node.EndByte() > uint32(len(source)) {
		return nil, true
	}
	base, _ := parseJavaTypeString(typ.Content(source))
	family := nativeFunctionalFamily(base, ctx)
	if family != "Function" && family != "Supplier" {
		return nil, false
	}
	c := &anonymousFunctionalCertificate{graph: symbol.GlobalScope, file: ctx.currentFile, allocation: node, typ: typ, body: body, lexical: ctx, source: bytes.Clone(source), sourceIdentity: anonymousReceiverSourceIdentity(source), family: family, phase: anonymousFunctionalBody}
	c.self = c
	if ctx.currentFile != nil {
		c.root = ctx.currentFile.ParsedRoot()
	}
	if ctx.currentClass != nil && ctx.currentClass.Class != nil {
		c.ownerNode = ctx.currentClass.Class.DeclarationNode
	}
	if ctx.localScope != nil {
		c.executable = ctx.localScope.DeclarationNode
	}
	if ctx.currentFile == nil || anonymousReceiverSourceIdentity(source) != anonymousReceiverSourceIdentity(ctx.currentFile.Source) || !anonymousFunctionalLexicalProof(c) {
		return nil, true
	}
	// An actual lexical TYPE binder denotes an ordinary non-native header.
	// This is not a fallback for invalid input/source/certificate ownership.
	if anonymousFunctionalOrdinaryPrefix(c.typ, c.lexical, c.source) {
		return nil, false
	}
	o, ok := anonymousFunctionalHeader(c)
	if !ok {
		return nil, true
	}
	c.arguments = o.arguments
	// Raw source syntax stays raw; only diamond acquires the actual target ABI.
	c.header = typ.Content(source)
	if strings.Contains(c.header, "<>") {
		args := make([]string, len(o.arguments))
		for i, a := range o.arguments {
			args[i] = anonymousFunctionalOriginText(a)
		}
		c.header = base + "<" + strings.Join(args, ",") + ">"
	}
	return c, true
}
func anonymousFunctionalOriginText(o inferredJavaTypeOrigin) string {
	s := o.externalName
	if o.parameter != nil {
		s = o.parameter.SourceName
	} else if o.nominalScope != nil {
		s = javaClassBinaryName(o.nominalScope)
	} else if s == "" {
		s = stripOriginArrayDisplay(o.javaType)
	}
	if len(o.arguments) > 0 {
		a := make([]string, len(o.arguments))
		for i, v := range o.arguments {
			a[i] = anonymousFunctionalOriginText(v)
		}
		s += "<" + strings.Join(a, ",") + ">"
	}
	return s + strings.Repeat("[]", o.arrayRank)
}

// Physical emission consumes already-proved origins. External mapping receives
// fresh internal slots rather than reparsing argument names in the body namespace.
func anonymousFunctionalPhysical(o inferredJavaTypeOrigin, ctx Ctx) (ast.Expr, bool) {
	if !completeJavaTypeOrigin(o) || o.wildcard {
		return nil, false
	}
	if o.parameter != nil || o.nominalScope != nil || o.arrayRank != 0 {
		return classCastResultGoType(o, ctx)
	}
	if canonical, ok := canonicalResultPhysicalExpr(o, ctx); ok {
		return canonical, true
	}
	if o.externalName == "" {
		return nil, false
	}
	mapping := ctx.Clone()
	mapping.currentClass = nil
	mapping.currentFile = nil
	mapping.localScope = nil
	mapping.syntheticTypeParameters = nil
	mapping.importAliases = map[string]string{}
	mapping.usedImports = map[string]bool{}
	slots := map[string]ast.Expr{}
	names := make([]string, len(o.arguments))
	for i, a := range o.arguments {
		value, ok := anonymousFunctionalPhysical(a, ctx)
		if !ok {
			return nil, false
		}
		names[i] = "__java2goAnonymousSlot" + strconv.Itoa(i)
		slots[names[i]] = value
	}
	text := o.externalName
	if len(names) > 0 {
		text += "<" + strings.Join(names, ",") + ">"
	}
	value := javaTypeStringToGoTypeExpr(text, names, mapping)
	return staticFieldNativeStorageExpr(value, mapping.importAliases[stdjavaImportPath], slots, ctx)
}

func anonymousFunctionalMethodOrigin(c *anonymousFunctionalCertificate, m *symbol.Definition, t string) (inferredJavaTypeOrigin, bool) {
	ctx := c.lexical.Clone()
	ctx.currentClass = c.scope
	ctx.localScope = m
	base, _ := parseJavaTypeString(t)
	prefix := strings.Split(base, ".")[0]
	for _, member := range nodeutil.NamedChildrenOf(c.body) {
		switch member.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			if name := member.ChildByFieldName("name"); name != nil && name.Content(c.source) == prefix {
				return inferredJavaTypeOrigin{}, false
			}
		}
	}
	o := anonymousFunctionalOrigin(t, ctx)
	return o, anonymousFunctionalOriginOwned(o, ctx)
}
func anonymousFunctionalSelected(c *anonymousFunctionalCertificate) (*symbol.Definition, inferredJavaTypeOrigin, bool) {
	if c.scope == nil {
		return nil, inferredJavaTypeOrigin{}, false
	}
	name := "get"
	arity := 0
	if c.family == "Function" {
		name = "apply"
		arity = 1
	}
	var selected *symbol.Definition
	var result inferredJavaTypeOrigin
	for _, m := range c.scope.Methods {
		if m == nil || m.OriginalName != name {
			continue
		}
		n := m.DeclarationNode
		if n == nil || n.Type() != "method_declaration" || !anonymousFunctionalActualChild(c.body, n) || !declarationHasModifier(n, "public") || declarationHasModifier(n, "static") || declarationHasModifier(n, "private") || declarationHasModifier(n, "abstract") || declarationHasModifier(n, "native") || declarationHasModifier(n, "default") || n.ChildByFieldName("body") == nil || n.ChildByFieldName("type_parameters") != nil {
			continue
		}
		formals := nodeutil.SemanticNamedChildrenOf(n.ChildByFieldName("parameters"))
		if len(formals) != arity {
			continue
		}
		if m.Constructor || m.IsStatic || m.IsPrivate || m.IsVariadic || m.RuntimeDefault || m.RequiresHelper || m.HelperName != "" || !m.HasBody || len(m.TypeParameters) != 0 || len(m.Parameters) != arity || n.ChildByFieldName("name") == nil || n.ChildByFieldName("name").Content(c.source) != m.OriginalName || n.ChildByFieldName("type") == nil || n.ChildByFieldName("type").Content(c.source) != m.OriginalType {
			return nil, result, false
		}
		matches := true
		for i, p := range formals {
			if p.Type() != "formal_parameter" || p.ChildByFieldName("type") == nil || p.ChildByFieldName("name") == nil || m.Parameters[i] == nil || m.Parameters[i].OriginalType != p.ChildByFieldName("type").Content(c.source) || m.Parameters[i].OriginalName != p.ChildByFieldName("name").Content(c.source) {
				return nil, result, false
			}
			origin, ok := anonymousFunctionalMethodOrigin(c, m, p.ChildByFieldName("type").Content(c.source))
			stored := declaredJavaTypeOrigin(definitionDeclaredJavaType(m.Parameters[i]), func() Ctx { x := c.lexical; x.currentClass = c.scope; x.localScope = m; return x }())
			matches = matches && ok && sameJavaTypeOrigin(origin, c.arguments[i]) && sameJavaTypeOrigin(origin, stored)
		}
		if !matches {
			continue
		}
		actual, ok := anonymousFunctionalMethodOrigin(c, m, m.OriginalType)
		resultCtx := c.lexical.Clone()
		resultCtx.currentClass = c.scope
		resultCtx.localScope = m
		storedResult := declaredJavaTypeOrigin(definitionDeclaredJavaType(m), resultCtx)
		if !ok || !sameJavaTypeOrigin(actual, storedResult) {
			return nil, result, false
		}
		desired := c.arguments[len(c.arguments)-1]
		if !sameJavaTypeOrigin(actual, desired) {
			_, _, applicable, handled := declaredOriginConversion(actual, desired, c.lexical)
			if !handled || !applicable {
				continue
			}
		}
		if selected != nil {
			return nil, result, false
		}
		selected = m
		result = actual
	}
	return selected, result, selected != nil
}
func anonymousFunctionalAttach(c *anonymousFunctionalCertificate, info *anonymousClassInfo, ctx Ctx) bool {
	c.info = info
	c.scope = info.scope
	c.class = info.scope.Class
	c.hoistStart = len(*ctx.hoistedDecls)
	info.functional = c
	info.scope.DeclaredTypeParameters = []symbol.TypeParam{}
	bindings := symbol.VisibleTypeParamBindings(info.scope.TypeParameters)
	for _, m := range info.scope.Methods {
		if m == nil {
			continue
		}
		m.TypeParameterBindings = bindings
		m.DirectTypeParameter = symbol.DirectTypeParamForJavaType(m.OriginalType, info.scope.TypeParameters)
		for _, p := range m.Parameters {
			if p != nil {
				p.TypeParameterBindings = bindings
				p.DirectTypeParameter = symbol.DirectTypeParamForJavaType(p.OriginalType, info.scope.TypeParameters)
			}
		}
	}
	method, result, ok := anonymousFunctionalSelected(c)
	if !ok {
		c.phase = anonymousFunctionalFailed
		return false
	}
	c.selected = method
	if !sameJavaTypeOrigin(result, c.arguments[len(c.arguments)-1]) {
		used := map[string]struct{}{}
		for _, m := range info.scope.Methods {
			used[m.Name] = struct{}{}
		}
		method.Name = nextSyntheticMemberName(method.Name, used)
	}
	c.shape = anonymousFunctionalShape(info)
	c.lexicalBindings = anonymousFunctionalBindings(visibleTypeParameterDeclarations(c.lexical))
	return true
}
func anonymousFunctionalValid(c *anonymousFunctionalCertificate, ctx Ctx, complete bool) bool {
	if c == nil || c.self != c || c.phase == anonymousFunctionalFailed || c.phase == anonymousFunctionalBody && c.sealed || complete && c.phase != anonymousFunctionalComplete || c.info == nil || c.info.functional != c || c.info.scope != c.scope || c.scope == nil || c.scope.Class != c.class || c.scope.Enclosing != c.lexical.currentClass || !anonymousFunctionalEqual(c.scope.Class.DeclarationNode, c.allocation) || !anonymousFunctionalLexicalProof(c) {
		return false
	}
	key, ok := anonymousClassSourceKey(c.allocation)
	if !ok || ctx.anonymousClasses[key] != c.info || ctx.currentFile != c.file || ctx.hoistedDecls != c.lexical.hoistedDecls || ctx.anonClassCounter != c.lexical.anonClassCounter || ctx.genericFamilies != c.lexical.genericFamilies || ctx.dependentTypeWitnesses != c.lexical.dependentTypeWitnesses {
		return false
	}
	header, ok := anonymousFunctionalHeader(c)
	if !ok || len(header.arguments) != len(c.arguments) {
		return false
	}
	for i, o := range header.arguments {
		if !sameJavaTypeOrigin(o, c.arguments[i]) {
			return false
		}
	}
	if anonymousReceiverPhysicalABI(anonymousFunctionalRoleContext(c, ctx)) != anonymousReceiverPhysicalABI(c.lexical) {
		return false
	}
	method, _, ok := anonymousFunctionalSelected(c)
	if !ok || method != c.selected {
		return false
	}
	if c.phase == anonymousFunctionalComplete {
		if !anonymousFunctionalImplementation(c, ctx) || !anonymousFunctionalCaptureProof(c) || c.shape != anonymousFunctionalShape(c.info) || c.lexicalBindings != anonymousFunctionalBindings(visibleTypeParameterDeclarations(c.lexical)) || !c.info.emissionStampValid || c.info.emissionStamp != anonymousReceiverStamp(c.file.Source, c.lexical) {
			return false
		}
		if ctx.hoistedDecls == nil || len(*ctx.hoistedDecls) < c.hoistEnd || len(c.emitted) != len(c.emittedShape) {
			return false
		}
		for i, d := range c.emitted {
			if (*ctx.hoistedDecls)[c.hoistStart+i] != d || anonymousFunctionalAST(d) != c.emittedShape[i] {
				return false
			}
		}
	}
	return true
}
func anonymousFunctionalContract(c *anonymousFunctionalCertificate) sourceNativeFunctionalContract {
	args := make([]string, len(c.arguments))
	for i, a := range c.arguments {
		args[i] = anonymousFunctionalOriginText(a)
	}
	contract, _ := nativeFunctionalContract(c.family, args)
	return contract
}
func anonymousFunctionalContractEqual(c *anonymousFunctionalCertificate, contract sourceNativeFunctionalContract) bool {
	own := anonymousFunctionalContract(c)
	return own.family == contract.family && own.method == contract.method && own.result == contract.result && strings.Join(own.arguments, "\x00") == strings.Join(contract.arguments, "\x00") && strings.Join(own.parameters, "\x00") == strings.Join(contract.parameters, "\x00")
}
func anonymousFunctionalView(c *anonymousFunctionalCertificate, receiver ast.Expr, ctx Ctx) ast.Expr {
	if !anonymousFunctionalValid(c, ctx, true) || receiver == nil {
		return nil
	}
	method, result, ok := anonymousFunctionalSelected(c)
	if !ok {
		return nil
	}
	params := &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}
	args := []ast.Expr{ast.NewIdent("execution")}
	for i := 0; i < len(c.arguments)-1; i++ {
		typ, ok := anonymousFunctionalPhysical(c.arguments[i], ctx)
		if !ok {
			return nil
		}
		name := "value" + strconv.Itoa(i)
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(name)}, Type: typ})
		args = append(args, ast.NewIdent(name))
	}
	output, ok := anonymousFunctionalPhysical(c.arguments[len(c.arguments)-1], ctx)
	if !ok {
		return nil
	}
	call := ast.Expr(&ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(executionImplementationName(method, c.scope, ctx))}, Args: args})
	if !sameJavaTypeOrigin(result, c.arguments[len(c.arguments)-1]) {
		descriptor, ok := declaredOriginDescriptor(c.arguments[len(c.arguments)-1], ctx, map[*symbol.TypeParamDeclaration]bool{})
		if !ok {
			return nil
		}
		call = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{output}, []ast.Expr{call, descriptor})
	}
	closure := &ast.FuncLit{Type: &ast.FuncType{Params: params, Results: &ast.FieldList{List: []*ast.Field{{Type: output}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}}
	types := make([]ast.Expr, len(c.arguments))
	for i, a := range c.arguments {
		types[i], ok = anonymousFunctionalPhysical(a, ctx)
		if !ok {
			return nil
		}
	}
	return stdjavaGenericCall(ctx, "New"+c.family+"SourceView", types, []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent("ObjectInfo")}, closure})
}

// Complete only after bodies/initializers were emitted. Identity generation is
// intentionally deferred until then, so incomplete info never exports a view.
func anonymousFunctionalFinish(c *anonymousFunctionalCertificate, ctx Ctx) bool {
	if c.sealed || c.phase != anonymousFunctionalBody || !anonymousFunctionalValid(c, ctx, false) || !anonymousFunctionalImplementation(c, ctx) || !anonymousFunctionalCaptureProof(c) || c.shape != anonymousFunctionalShape(c.info) || c.lexicalBindings != anonymousFunctionalBindings(visibleTypeParameterDeclarations(c.lexical)) {
		return false
	}
	c.shape = anonymousFunctionalShape(c.info)
	c.lexicalBindings = anonymousFunctionalBindings(visibleTypeParameterDeclarations(c.lexical))
	c.phase = anonymousFunctionalComplete
	return true
}
func anonymousFunctionalSeal(c *anonymousFunctionalCertificate, ctx Ctx) {
	c.sealed = true
	c.hoistEnd = len(*ctx.hoistedDecls)
	c.emitted = append([]ast.Decl(nil), (*ctx.hoistedDecls)[c.hoistStart:c.hoistEnd]...)
	for _, d := range c.emitted {
		c.emittedShape = append(c.emittedShape, anonymousFunctionalAST(d))
	}
}
func anonymousFunctionalIdentity(c *anonymousFunctionalCertificate, ctx Ctx) []ast.Decl {
	if !anonymousFunctionalValid(c, ctx, true) {
		return nil
	}
	id := javaClassBinaryName(c.lexical.currentClass) + "$" + c.info.structName
	decls := syntheticHierarchicalReferenceIdentityDecls(c.info.structName, id, c.scope, nil, nil, c.scope.GoTypeParameterNames(), ctx)
	if len(decls) != 3 {
		return nil
	}
	view := decls[1].(*ast.FuncDecl)
	name := view.Recv.List[0].Names[0].Name
	native := anonymousFunctionalView(c, ast.NewIdent(name), ctx)
	if native == nil {
		return nil
	}
	cases := view.Body.List[0].(*ast.SwitchStmt).Body
	clause := &ast.CaseClause{List: []ast.Expr{javaTypeIDLiteral("java.util.function."+c.family, ctx)}, Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{native}}}}
	cases.List = append(cases.List[:len(cases.List)-1], clause, cases.List[len(cases.List)-1])
	return decls
}
func anonymousFunctionalBridges(c *anonymousFunctionalCertificate, ctx Ctx) []ast.Decl {
	if !anonymousFunctionalValid(c, ctx, true) {
		return nil
	}
	method, result, ok := anonymousFunctionalSelected(c)
	if !ok {
		return nil
	}
	name := "Get"
	if c.family == "Function" {
		name = "Apply"
	}
	var declarations []ast.Decl
	for _, aware := range []bool{false, true} {
		selector := name
		if aware {
			selector += executionMethodSuffix
		}
		implementation := executionImplementationName(method, c.scope, ctx)
		if sameJavaTypeOrigin(result, c.arguments[len(c.arguments)-1]) && ((!aware && method.Name == selector) || (aware && implementation == selector)) {
			continue
		}
		receiver := ast.NewIdent("receiver")
		execution := ast.Expr(stdjavaCall(ctx, "NewExecution"))
		params := &ast.FieldList{}
		if aware {
			execution = ast.NewIdent("execution")
			params.List = append(params.List, executionParameterField("execution", ctx))
		}
		args := []ast.Expr{execution}
		for i := 0; i < len(c.arguments)-1; i++ {
			typ, ok := anonymousFunctionalPhysical(c.arguments[i], ctx)
			if !ok {
				return nil
			}
			arg := "value" + strconv.Itoa(i)
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(arg)}, Type: typ})
			args = append(args, ast.NewIdent(arg))
		}
		output, ok := anonymousFunctionalPhysical(c.arguments[len(c.arguments)-1], ctx)
		if !ok {
			return nil
		}
		call := ast.Expr(&ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(implementation)}, Args: args})
		if !sameJavaTypeOrigin(result, c.arguments[len(c.arguments)-1]) {
			descriptor, ok := declaredOriginDescriptor(c.arguments[len(c.arguments)-1], ctx, map[*symbol.TypeParamDeclaration]bool{})
			if !ok {
				return nil
			}
			call = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{output}, []ast.Expr{call, descriptor})
		}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(selector), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: &ast.StarExpr{X: instantiateGenericType(c.info.structName, typeParamExprs(c.scope.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: params, Results: &ast.FieldList{List: []*ast.Field{{Type: output}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}})
	}
	return declarations
}

// Normalize only the declaration lookup role. Raw generic/erased body/lambda
// physical ABI inputs remain the consumer's inputs and must match the source.
func anonymousFunctionalRoleContext(c *anonymousFunctionalCertificate, ctx Ctx) Ctx {
	ctx.currentClass = c.lexical.currentClass
	ctx.localScope = c.lexical.localScope
	// The consumer's expected result is not the implementation's physical ABI.
	// Its actual emitted signature is independently checked above.
	ctx.lastType = c.lexical.lastType
	return ctx
}
func anonymousFunctionalConstructionAttempt(info *anonymousClassInfo, typ *sitter.Node, source []byte, ctx Ctx) bool {
	if info.functional != nil {
		return true
	}

	written := []string{}
	if typ != nil {
		written = append(written, typ.Content(source))
	}
	if info.scope != nil {
		written = append(written, info.scope.ImplementedInterfaces...)
	}
	for _, edge := range written {
		family := nativeFunctionalFamily(edge, ctx)
		if family == "Function" || family == "Supplier" {
			return true
		}
	}
	return false
}
func anonymousFunctionalCaptureProof(c *anonymousFunctionalCertificate) bool {
	fresh := collectCapturedLocals(c.body, c.source, c.lexical)
	if len(fresh) != len(c.info.captured) {
		return false
	}
	for i, current := range c.info.captured {
		actual := fresh[i]
		if current.name != actual.name || current.javaDef == nil || actual.javaDef == nil || current.javaDef.OriginalName != actual.javaDef.OriginalName || current.javaDef.OriginalType != actual.javaDef.OriginalType || current.javaDef.DirectTypeParameter != actual.javaDef.DirectTypeParameter || anonymousReceiverTypeBindings(current.javaDef.TypeParameterBindings) != anonymousReceiverTypeBindings(actual.javaDef.TypeParameterBindings) || anonymousFunctionalAST(current.goType) != anonymousFunctionalAST(actual.goType) {
			return false
		}
	}
	return true
}

// Require the selected implementation's actual emitted receiver/Execution/ABI;
// a completed descriptor cannot be donated to a different physical body.
func anonymousFunctionalImplementation(c *anonymousFunctionalCertificate, ctx Ctx) bool {
	if ctx.hoistedDecls == nil || c.selected == nil || len(*ctx.hoistedDecls) < c.hoistStart {
		return false
	}
	method, result, ok := anonymousFunctionalSelected(c)
	if !ok {
		return false
	}
	own := ctx.Clone()
	own.currentClass = c.scope
	own.localScope = method
	implementation := executionImplementationName(method, c.scope, own)
	found := false
	initializer := c.info.fieldInitializerMethodName == ""
	for _, d := range (*ctx.hoistedDecls)[c.hoistStart:] {
		f, ok := d.(*ast.FuncDecl)
		if !ok || f.Recv == nil || len(f.Recv.List) != 1 || f.Name == nil || f.Type == nil || f.Body == nil {
			continue
		}
		receiver, ok := f.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		expected := instantiateGenericType(c.info.structName, typeParamExprs(c.scope.GoTypeParameterNames()))
		if anonymousFunctionalAST(receiver.X) != anonymousFunctionalAST(expected) {
			continue
		}
		if f.Name.Name == c.info.fieldInitializerMethodName+executionMethodSuffix {
			initializer = true
		}
		if f.Name.Name != implementation {
			continue
		}
		if found || f.Type.TypeParams != nil || f.Type.Params == nil || len(f.Type.Params.List) != len(c.arguments) || f.Type.Results == nil || len(f.Type.Results.List) != 1 {
			return false
		}
		if anonymousFunctionalAST(f.Type.Params.List[0].Type) != anonymousFunctionalAST(executionParameterField("execution", own).Type) {
			return false
		}
		for i := 0; i < len(c.arguments)-1; i++ {
			physical, ok := anonymousFunctionalPhysical(c.arguments[i], own)
			if !ok || anonymousFunctionalAST(f.Type.Params.List[i+1].Type) != anonymousFunctionalAST(physical) {
				return false
			}
		}
		physical, ok := anonymousFunctionalPhysical(result, own)
		if !ok || anonymousFunctionalAST(f.Type.Results.List[0].Type) != anonymousFunctionalAST(physical) {
			return false
		}
		found = true
	}
	return found && initializer
}

// Retained import tables and the registered TYPE census must still match the
// actual file AST. S0 input validity alone does not establish either fact.
func anonymousFunctionalNamespace(file *symbol.FileScope) bool {
	if file == nil || file.ParsedRoot() == nil || file.ParsedRoot().HasError() || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil {
		return false
	}
	imports := map[string]string{}
	pkg := ""
	var declarations []*sitter.Node
	for _, n := range nodeutil.SemanticNamedChildrenOf(file.ParsedRoot()) {
		switch n.Type() {
		case "package_declaration":
			if n.NamedChild(0) == nil {
				return false
			}
			pkg = n.NamedChild(0).Content(file.Source)
		case "import_declaration":
			declaration, known := nodeutil.ReadJavaImportSyntax(n, file.Source)
			if !known {
				return false
			}
			if declaration.Kind != nodeutil.JavaImportSingleType {
				continue
			}
			name, path := declaration.Name, declaration.Scope
			if old, ok := imports[name]; ok && old != path {
				return false
			}
			imports[name] = path
		case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
			declarations = append(declarations, n)
		}
	}
	if pkg != file.Package || len(imports) != len(file.Imports) || len(declarations) != len(file.TopLevelClasses) {
		return false
	}
	for name, path := range imports {
		if file.Imports[name] != path {
			return false
		}
	}
	var matches func(*symbol.ClassScope, *sitter.Node) bool
	matches = func(scope *symbol.ClassScope, n *sitter.Node) bool {
		if scope == nil || scope.Class == nil || !anonymousFunctionalEqual(scope.Class.DeclarationNode, n) || n.ChildByFieldName("name") == nil || scope.Class.OriginalName != n.ChildByFieldName("name").Content(file.Source) {
			return false
		}
		var nested []*sitter.Node
		if body := n.ChildByFieldName("body"); body != nil {
			for _, member := range nodeutil.SemanticNamedChildrenOf(body) {
				switch member.Type() {
				case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
					nested = append(nested, member)
				}
			}
		}
		if len(nested) != len(scope.Subclasses) {
			return false
		}
		for i, child := range nested {
			if !matches(scope.Subclasses[i], child) {
				return false
			}
		}
		return true
	}
	for i, n := range declarations {
		if !matches(file.TopLevelClasses[i], n) {
			return false
		}
	}
	return true
}

// TYPE declarations in enclosing class bodies are visible throughout the body;
// local types in an executable block become visible at their source position.
// A missing/forged resolver entry cannot turn one of those prefixes into JDK.
func anonymousFunctionalPrefixOwned(text string, node *sitter.Node, ctx Ctx, source []byte) bool {
	base, args := parseJavaTypeString(text)
	base, _ = javaArrayTypeParts(base)
	first := strings.Split(base, ".")[0]
	for p := node.Parent(); p != nil; p = p.Parent() {
		classBody := p.Type() == "class_body" || p.Type() == "interface_body" || p.Type() == "enum_body"
		block := p.Type() == "block" || p.Type() == "constructor_body"
		if !classBody && !block {
			continue
		}
		for _, declaration := range nodeutil.SemanticNamedChildrenOf(p) {
			switch declaration.Type() {
			case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
				if !classBody && declaration.StartByte() > node.StartByte() {
					continue
				}
				name := declaration.ChildByFieldName("name")
				if name == nil || name.Content(source) != first {
					continue
				}
				resolved := resolveClassScopeByQualifiedName(ctx, first)
				if resolved == nil || resolved.Class == nil || !anonymousFunctionalEqual(resolved.Class.DeclarationNode, declaration) {
					return false
				}
			}
		}
	}
	for _, arg := range args {
		if !anonymousFunctionalPrefixOwned(arg, node, ctx, source) {
			return false
		}
	}
	return true
}

func anonymousFunctionalExecutable(c *anonymousFunctionalCertificate) bool {
	m := c.lexical.localScope
	if c.executable == nil {
		return m == nil
	}
	n := c.executable
	if m == nil || n.ChildByFieldName("name") == nil || m.OriginalName != n.ChildByFieldName("name").Content(c.source) || m.Constructor != (n.Type() == "constructor_declaration") || m.IsStatic != declarationHasModifier(n, "static") || m.IsPrivate != declarationHasModifier(n, "private") || m.IsFinal != declarationHasModifier(n, "final") || m.RuntimeDefault || m.HasBody != (n.ChildByFieldName("body") != nil) {
		return false
	}
	if n.Type() == "method_declaration" {
		if typ := n.ChildByFieldName("type"); typ == nil || typ.Content(c.source) != m.OriginalType {
			return false
		}
	}
	params := nodeutil.SemanticNamedChildrenOf(n.ChildByFieldName("parameters"))
	if len(params) != len(m.Parameters) {
		return false
	}
	expectedBindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(c.lexical))
	for i, p := range params {
		typ, name := nodeutil.JavaParameterNodes(p)
		if typ == nil || name == nil || m.Parameters[i] == nil || typ.Content(c.source) != m.Parameters[i].OriginalType || name.Content(c.source) != m.Parameters[i].OriginalName {
			return false
		}
		actual := declaredJavaTypeOrigin(symbol.JavaType{Original: typ.Content(c.source), TypeParameterBindings: expectedBindings}, c.lexical)
		stored := declaredJavaTypeOrigin(definitionDeclaredJavaType(m.Parameters[i]), c.lexical)
		if !sameJavaTypeOrigin(actual, stored) {
			return false
		}
	}
	return true
}

// A lexical role is deliberately separate from functional completion. Ordinary
// anonymous owners (for example Runnable) can contain named declarations too.
// This record can prove namespace/body ownership; it cannot grant any family,
// construction, selected SAM, native view or completed emission stamp.
type anonymousLexicalOwnerCertificate struct {
	self                        *anonymousLexicalOwnerCertificate
	info                        *anonymousClassInfo
	scope                       *symbol.ClassScope
	class                       *symbol.Definition
	graph                       *symbol.GlobalSymbols
	file                        *symbol.FileScope
	root, allocation, typ, body *sitter.Node
	lexical                     Ctx
	source                      []byte
	backing                     *byte
	phase                       uint8
	shape, binders              string
}

func anonymousLexicalOwnerShape(info *anonymousClassInfo) string {
	if info == nil || info.scope == nil || info.scope.Class == nil {
		return "invalid"
	}
	s := info.scope
	var out strings.Builder
	fmt.Fprintf(&out, "%p:%p:%p:%p:%q:%q:%s:%s|", info, s, s.Class, s.Enclosing, s.Superclass, s.ImplementedInterfaces, anonymousFunctionalBindings(s.TypeParameters), anonymousFunctionalBindings(s.DeclaredTypeParameters))
	out.WriteString(anonymousFunctionalDefinition(s.Class))
	for _, field := range s.Fields {
		out.WriteString(anonymousFunctionalDefinition(field))
	}
	for _, method := range s.Methods {
		out.WriteString(anonymousFunctionalDefinition(method))
	}
	for _, capture := range info.captured {
		fmt.Fprintf(&out, "%q:%q:%s:%s|", capture.name, capture.fieldName, anonymousFunctionalAST(capture.goType), anonymousFunctionalDefinition(capture.javaDef))
	}
	return out.String()
}

// Identify the actual class-body boundary. A local declaration in an anonymous
// method belongs to the object creation owning that body, not to a distant Main.
func anonymousLexicalNearestOwner(node *sitter.Node) (*sitter.Node, *sitter.Node) {
	var executable *sitter.Node
	for current := node.Parent(); current != nil; current = current.Parent() {
		if executable == nil && (current.Type() == "method_declaration" || current.Type() == "constructor_declaration") {
			executable = current
		}
		if current.Type() == "class_body" {
			allocation := current.Parent()
			if allocation != nil && allocation.Type() == "object_creation_expression" && anonymousFunctionalEqual(objectCreationClassBody(allocation), current) {
				return allocation, executable
			}
		}
		switch current.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
			return current, executable
		}
	}
	return nil, executable
}

func anonymousLexicalDeclaredMethod(scope *symbol.ClassScope, body, node *sitter.Node, source []byte) bool {
	if scope == nil || body == nil || node == nil || !anonymousFunctionalActualChild(body, node) {
		return false
	}
	var method *symbol.Definition
	for _, candidate := range scope.Methods {
		if candidate != nil && anonymousFunctionalEqual(candidate.DeclarationNode, node) {
			if method != nil {
				return false
			}
			method = candidate
		}
	}
	if method == nil || node.ChildByFieldName("name") == nil || method.OriginalName != node.ChildByFieldName("name").Content(source) || method.Constructor != (node.Type() == "constructor_declaration") || method.IsStatic != declarationHasModifier(node, "static") || method.IsPrivate != declarationHasModifier(node, "private") || method.HasBody != (node.ChildByFieldName("body") != nil) || method.RuntimeDefault || !anonymousFunctionalDeclaredBinders(node, method.TypeParameters, source) {
		return false
	}
	if node.Type() == "method_declaration" {
		typ := node.ChildByFieldName("type")
		if typ == nil || typ.Content(source) != method.OriginalType {
			return false
		}
	}
	parameters := nodeutil.SemanticNamedChildrenOf(node.ChildByFieldName("parameters"))
	if len(parameters) != len(method.Parameters) {
		return false
	}
	for i, p := range parameters {
		typ, name := nodeutil.JavaParameterNodes(p)
		if typ == nil || name == nil || method.Parameters[i] == nil || method.Parameters[i].OriginalName != name.Content(source) || method.Parameters[i].OriginalType != typ.Content(source) {
			return false
		}
	}
	return true
}

// Return an independently proved lexical body, without asking an anonymous
// owner to become named or to export its native functional identity.
func anonymousLexicalOwnerFrame(scope *symbol.ClassScope, ctx Ctx, active map[*symbol.ClassScope]bool) (*symbol.FileScope, *sitter.Node, *sitter.Node, bool) {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil, nil, nil, false
	}
	if scope.HasAnonymousSourceOrigin() {
		var found *anonymousLexicalOwnerCertificate
		for key, info := range ctx.anonymousClasses {
			if info == nil || info.scope != scope {
				continue
			}
			role := info.lexicalOwner
			if role == nil || role.self != role || role.info != info || role.scope != scope || found != nil {
				return nil, nil, nil, false
			}
			original, ok := anonymousClassSourceKey(role.allocation)
			if !ok || original != key {
				return nil, nil, nil, false
			}
			found = role
		}
		if found == nil || !anonymousLexicalOwnerValid(found, ctx, active) {
			return nil, nil, nil, false
		}
		return found.file, found.allocation, found.body, true
	}
	if !anonymousFunctionalActualNamedSeen(scope, ctx, active) {
		return nil, nil, nil, false
	}
	var file *symbol.FileScope
	for current := scope; current != nil && file == nil; current = current.Enclosing {
		file = findFileScopeForClassScope(current, ctx)
	}
	if file == nil || !anonymousFunctionalNamespace(file) {
		return nil, nil, nil, false
	}
	return file, scope.Class.DeclarationNode, scope.Class.DeclarationNode.ChildByFieldName("body"), true
}

func anonymousLexicalOwnerSource(role *anonymousLexicalOwnerCertificate, ctx Ctx, active map[*symbol.ClassScope]bool) bool {
	if role == nil || role.self != role || role.graph != symbol.GlobalScope || role.file == nil || ctx.currentFile != role.file || !anonymousFunctionalNamespace(role.file) || !anonymousFunctionalEqual(role.file.ParsedRoot(), role.root) || anonymousReceiverSourceIdentity(role.file.Source) != role.backing || !bytes.Equal(role.file.Source, role.source) || !anonymousFunctionalWithin(role.allocation, role.root) || !anonymousFunctionalEqual(role.allocation.ChildByFieldName("type"), role.typ) || !anonymousFunctionalActualChild(role.allocation, role.typ) || !anonymousFunctionalEqual(objectCreationClassBody(role.allocation), role.body) || !anonymousFunctionalActualChild(role.allocation, role.body) {
		return false
	}
	owner := role.lexical.currentClass
	file, anchor, body, owned := anonymousLexicalOwnerFrame(owner, role.lexical, active)
	if !owned || file != role.file {
		return false
	}
	nearest, executable := anonymousLexicalNearestOwner(role.allocation)
	if !anonymousFunctionalEqual(nearest, anchor) {
		return false
	}
	if executable == nil {
		if role.lexical.localScope != nil {
			return false
		}
	} else {
		if role.lexical.localScope == nil || !anonymousFunctionalEqual(role.lexical.localScope.DeclarationNode, executable) || !anonymousLexicalDeclaredMethod(owner, body, executable, role.source) {
			return false
		}
	}
	return true
}

func anonymousLexicalOwnerAttach(info *anonymousClassInfo, node, typ, body *sitter.Node, source []byte, ctx Ctx) *anonymousLexicalOwnerCertificate {
	if info == nil || info.scope == nil || info.scope.Class == nil || !info.scope.HasAnonymousSourceOrigin() || ctx.currentFile == nil || node == nil || typ == nil || body == nil || node.Type() != "object_creation_expression" || !anonymousFunctionalEqual(info.scope.Class.DeclarationNode, node) || anonymousReceiverSourceIdentity(source) != anonymousReceiverSourceIdentity(ctx.currentFile.Source) {
		return nil
	}
	role := &anonymousLexicalOwnerCertificate{info: info, scope: info.scope, class: info.scope.Class, graph: symbol.GlobalScope, file: ctx.currentFile, root: ctx.currentFile.ParsedRoot(), allocation: node, typ: typ, body: body, lexical: ctx, source: bytes.Clone(source), backing: anonymousReceiverSourceIdentity(source), phase: anonymousFunctionalBody}
	role.self = role
	role.shape = anonymousLexicalOwnerShape(info)
	role.binders = anonymousFunctionalBindings(visibleTypeParameterDeclarations(ctx))
	if !anonymousLexicalOwnerSource(role, ctx, map[*symbol.ClassScope]bool{}) {
		return nil
	}
	return role
}

func anonymousLexicalOwnerValid(role *anonymousLexicalOwnerCertificate, ctx Ctx, active map[*symbol.ClassScope]bool) bool {
	if role == nil || role.self != role || role.phase == anonymousFunctionalFailed || role.info == nil || role.info.lexicalOwner != role || role.info.scope != role.scope || role.scope == nil || !role.scope.HasAnonymousSourceOrigin() || role.scope.Class != role.class || role.scope.Enclosing != role.lexical.currentClass || !anonymousFunctionalEqual(role.scope.Class.DeclarationNode, role.allocation) || ctx.hoistedDecls != role.lexical.hoistedDecls || ctx.anonClassCounter != role.lexical.anonClassCounter || ctx.genericFamilies != role.lexical.genericFamilies || ctx.dependentTypeWitnesses != role.lexical.dependentTypeWitnesses {
		return false
	}
	if active[role.scope] {
		return false
	}
	active[role.scope] = true
	defer delete(active, role.scope)
	key, ok := anonymousClassSourceKey(role.allocation)
	if !ok || ctx.anonymousClasses[key] != role.info || role.shape != anonymousLexicalOwnerShape(role.info) || role.binders != anonymousFunctionalBindings(visibleTypeParameterDeclarations(role.lexical)) || !anonymousLexicalOwnerSource(role, ctx, active) {
		return false
	}
	if role.phase == anonymousFunctionalComplete && (!role.info.emissionStampValid || role.info.emissionStamp != anonymousReceiverStamp(role.file.Source, role.lexical)) {
		return false
	}
	return true
}

// True only for a declaration-bound first-segment TYPE binder in the actual
// lexical class/executable. Synthetic/caller display parameters do not qualify.
func anonymousFunctionalOrdinaryPrefix(typ *sitter.Node, ctx Ctx, source []byte) bool {
	if typ == nil || typ.EndByte() > uint32(len(source)) {
		return false
	}
	base, _ := parseJavaTypeString(typ.Content(source))
	first := strings.Split(base, ".")[0]
	binding := visibleTypeParameterDeclarationForJavaType(first, ctx)
	if binding == nil || binding.SourceName != first {
		return false
	}
	var actual []symbol.TypeParam
	if ctx.currentClass != nil && (ctx.localScope == nil || !ctx.localScope.IsStatic) {
		actual = symbol.MergeTypeParams(actual, ctx.currentClass.TypeParameters)
	}
	if ctx.localScope != nil {
		actual = symbol.MergeTypeParams(actual, ctx.localScope.TypeParameters)
	}
	return symbol.VisibleTypeParamBindings(actual)[first] == binding
}

// Ordinary first construction may run before role completion, but only for the
// exact original caller source/AST/lexical emission. Role-only context is not a
// transferable constructor permit and never grants native family authority.
func anonymousFunctionalOrdinaryConstruction(info *anonymousClassInfo, node, typ *sitter.Node, source []byte, ctx Ctx) (bool, bool) {
	if info == nil || info.functional != nil || info.lexicalOwner == nil {
		return false, false
	}
	role := info.lexicalOwner
	if !anonymousFunctionalOrdinaryPrefix(role.typ, role.lexical, role.source) {
		return false, false
	}
	if role.info != info || role.scope != info.scope || !anonymousLexicalOwnerValid(role, ctx, map[*symbol.ClassScope]bool{}) || !anonymousFunctionalEqual(node, role.allocation) || !anonymousFunctionalEqual(typ, role.typ) || !anonymousFunctionalEqual(node.ChildByFieldName("type"), typ) || !anonymousFunctionalEqual(objectCreationClassBody(node), role.body) || !anonymousFunctionalWithin(node, role.root) || anonymousReceiverSourceIdentity(source) != role.backing || !bytes.Equal(source, role.source) || ctx.currentFile != role.file || ctx.currentClass != role.lexical.currentClass || ctx.localScope != role.lexical.localScope || anonymousReceiverStamp(source, ctx) != anonymousReceiverStamp(role.file.Source, role.lexical) {
		return true, false
	}
	return true, true
}

// New-route eligibility is pure and does not replace named-family algorithms.
// The private marker excludes dedicated anonymous providers; it never proves
// namedness. Source selectors remain the existing allocated named-method ABI.
func anonymousFunctionalNamedViewBindings(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.HasAnonymousSourceOrigin() || !anonymousFunctionalActualNamed(scope, ctx) || ctx.currentFile == nil || ctx.currentFile.ParsedRoot() == nil || ctx.currentFile.ParsedRoot().HasChanges() || scope.DeclaredTypeParameters == nil || !anonymousFunctionalDeclaredBinders(scope.Class.DeclarationNode, scope.DeclaredTypeParameters, ctx.currentFile.Source) || len(scope.TypeParameters) < len(scope.DeclaredTypeParameters) {
		return false
	}
	declarationKind := scope.Class.DeclarationNode.Type()
	actualInterface := declarationKind == "interface_declaration" || declarationKind == "annotation_type_declaration"
	if scope.IsInterface != actualInterface || scope.IsEnum != (declarationKind == "enum_declaration") {
		return false
	}
	declarations := map[*symbol.TypeParamDeclaration]bool{}
	names := map[string]bool{}
	for _, parameter := range scope.TypeParameters {
		if parameter.Declaration == nil || parameter.Name != parameter.Declaration.SourceName || !token.IsIdentifier(parameter.EmittedName()) || declarations[parameter.Declaration] || names[parameter.EmittedName()] {
			return false
		}
		declarations[parameter.Declaration] = true
		names[parameter.EmittedName()] = true
	}
	prefix := len(scope.TypeParameters) - len(scope.DeclaredTypeParameters)
	for i, declared := range scope.DeclaredTypeParameters {
		carried := scope.TypeParameters[prefix+i]
		if carried.Declaration != declared.Declaration || anonymousFunctionalBindings([]symbol.TypeParam{carried}) != anonymousFunctionalBindings([]symbol.TypeParam{declared}) {
			return false
		}
	}
	var allowed []symbol.TypeParam
	if owner := scope.Enclosing; owner != nil {
		_, executable := anonymousLexicalNearestOwner(scope.Class.DeclarationNode)
		var method *symbol.Definition
		if executable != nil {
			for _, candidate := range owner.Methods {
				if candidate != nil && anonymousFunctionalEqual(candidate.DeclarationNode, executable) {
					if method != nil {
						return false
					}
					method = candidate
				}
			}
			file, _, body, owned := anonymousLexicalOwnerFrame(owner, ctx, map[*symbol.ClassScope]bool{})
			if !owned || file != ctx.currentFile || method == nil || !anonymousLexicalDeclaredMethod(owner, body, executable, file.Source) {
				return false
			}
		}
		if method == nil || !method.IsStatic {
			allowed = append(allowed, owner.TypeParameters...)
		}
		if method != nil {
			allowed = symbol.AppendTypeParamsByDeclaration(allowed, method.TypeParameters)
		}
	}
	// Hidden carriage retains the actual lexical declaration order, independently
	// of Java spelling shadowing. Own declarations form its exact trailing suffix.
	next := 0
	for _, carried := range scope.TypeParameters[:prefix] {
		found := false
		for next < len(allowed) {
			candidate := allowed[next]
			next++
			if candidate.Declaration == carried.Declaration {
				if anonymousFunctionalBindings([]symbol.TypeParam{candidate}) != anonymousFunctionalBindings([]symbol.TypeParam{carried}) {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func anonymousFunctionalNamedViewEligible(scope *symbol.ClassScope, contract sourceNativeFunctionalContract, ctx Ctx) bool {
	if !anonymousFunctionalNamedViewBindings(scope, ctx) {
		return false
	}
	selected, owner, _ := resolveSourceNativeSAM(scope, contract, ctx)
	ownerCtx := classScopeCtx(owner, ctx)
	if selected == nil || owner == nil || !anonymousFunctionalNamedViewBindings(owner, ownerCtx) {
		return false
	}
	node := selected.DeclarationNode
	source := ownerCtx.currentFile.Source
	if node == nil || node.Type() != "method_declaration" || node.HasChanges() || !anonymousLexicalDeclaredMethod(owner, owner.Class.DeclarationNode.ChildByFieldName("body"), node, source) || selected.Constructor || selected.RequiresHelper || selected.HelperName != "" || selected.RuntimeDefault || selected.IsVariadic || selected.IsStatic || selected.IsPrivate || !selected.HasBody || len(selected.TypeParameters) != 0 || !token.IsIdentifier(selected.Name) {
		return false
	}
	if (!owner.IsInterface && !declarationHasModifier(node, "public")) || declarationHasModifier(node, "static") || declarationHasModifier(node, "private") || declarationHasModifier(node, "abstract") || declarationHasModifier(node, "native") {
		return false
	}
	for _, other := range owner.Methods {
		if other != nil && other != selected && other.Name == selected.Name {
			return false
		}
	}
	// Source spread parameters use an array-reference ABI; IsVariadic is reserved
	// for platform metadata without AST. The actual signature helper supplies [].
	methodCtx := ownerCtx.Clone()
	methodCtx.localScope = selected
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(methodCtx))
	formals := nodeutil.SemanticNamedChildrenOf(node.ChildByFieldName("parameters"))
	if len(formals) != len(selected.Parameters) {
		return false
	}
	for i, formal := range formals {
		typ, _ := nodeutil.JavaParameterNodes(formal)
		if typ == nil || selected.Parameters[i] == nil {
			return false
		}
		parameter := selected.Parameters[i]
		if parameter.DirectTypeParameter != symbol.DirectTypeParamForJavaType(typ.Content(source), visibleTypeParameterDeclarations(methodCtx)) {
			return false
		}
		effective := definitionParameterJavaSignatureType(selected, i)
		fresh := declaredJavaTypeOrigin(symbol.JavaType{Original: effective, TypeParameterBindings: bindings}, methodCtx)
		stored := declaredJavaTypeOrigin(symbol.JavaType{Original: effective, TypeParameterBindings: parameter.TypeParameterBindings}, methodCtx)
		if !completeJavaTypeOrigin(fresh) || !sameJavaTypeOrigin(fresh, stored) {
			return false
		}
	}
	typ := node.ChildByFieldName("type")
	if typ == nil || selected.DirectTypeParameter != symbol.DirectTypeParamForJavaType(typ.Content(source), visibleTypeParameterDeclarations(methodCtx)) {
		return false
	}
	fresh := declaredJavaTypeOrigin(symbol.JavaType{Original: typ.Content(source), TypeParameterBindings: bindings}, methodCtx)
	stored := declaredJavaTypeOrigin(definitionDeclaredJavaType(selected), methodCtx)
	return completeJavaTypeOrigin(fresh) && sameJavaTypeOrigin(fresh, stored)
}
