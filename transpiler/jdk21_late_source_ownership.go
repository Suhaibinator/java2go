package transpiler

import (
	"bytes"
	"os"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A frame witnesses one actual convertFileNode call. It contains source roles
// and producer identities, never an admission result. The active pointer is a
// serial-render stack, like activeResolutionFiles; an outer context cannot use
// its token during a nested render, even for the same SourceFile.
type lateSourceOwnershipFrame struct {
	graph        *symbol.GlobalSymbols
	file         *symbol.FileScope
	packageName  string
	packageScope *symbol.PackageScope
	fileKey      string
	root         *sitter.Node
	snapshot     *symbol.SourceInputSnapshot
	invocation   *syntaxInvocation
	session      *cliownership.Session
	active       bool
	callerOwner  *symbol.ClassScope
	callerMethod *symbol.Definition
	owners       map[*symbol.ClassScope]lateSourceOwnerAnchor
	methods      map[*symbol.Definition]lateSourceMethodAnchor
	binders      map[*symbol.TypeParamDeclaration]lateSourceBinderAnchor
	late         map[*symbol.ClassScope]lateSourceRegistration
	anonymous    map[anonymousClassKey]*anonymousClassInfo
	locals       map[string]*localClassInfo
}

var activeLateSourceOwnership *lateSourceOwnershipFrame

type lateSourceBinderAnchor struct {
	node, owner *sitter.Node
	class       *symbol.ClassScope
	method      *symbol.Definition
	parameter   symbol.TypeParam
}

type lateSourceOwnerAnchor struct {
	class           *symbol.Definition
	javaName        string
	inner, iface    bool
	node            *sitter.Node
	enclosing       *symbol.ClassScope
	parameters, own []symbol.TypeParam
}

type lateSourceDefinitionAnchor struct {
	definition *symbol.Definition
	name, typ  string
	direct     *symbol.TypeParamDeclaration
	bindings   map[string]*symbol.TypeParamDeclaration
}

type lateSourceMethodAnchor struct {
	owner                              *symbol.ClassScope
	node                               *sitter.Node
	definition                         lateSourceDefinitionAnchor
	parameters                         []lateSourceDefinitionAnchor
	binders                            []symbol.TypeParam
	static, private, constructor, body bool
}

type lateSourceRegistration struct {
	producer   *symbol.ClassScope
	executable *symbol.Definition
	role       *sitter.Node
	anonymous  *anonymousClassInfo
	key        anonymousClassKey
	local      *localClassInfo
	name       string
}

func lateSourceCopyBindings(in map[string]*symbol.TypeParamDeclaration) map[string]*symbol.TypeParamDeclaration {
	if in == nil {
		return nil
	}
	out := make(map[string]*symbol.TypeParamDeclaration, len(in))
	for name, declaration := range in {
		out[name] = declaration
	}
	return out
}

func lateSourceCopyParameters(in []symbol.TypeParam) []symbol.TypeParam {
	out := append([]symbol.TypeParam(nil), in...)
	for i := range out {
		out[i].Bounds = append([]symbol.JavaType(nil), in[i].Bounds...)
		for j := range out[i].Bounds {
			out[i].Bounds[j].TypeParameterBindings = lateSourceCopyBindings(in[i].Bounds[j].TypeParameterBindings)
		}
	}
	return out
}

func lateSourceParametersMatch(actual, expected []symbol.TypeParam) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, p := range actual {
		q := expected[i]
		if p.Declaration != q.Declaration || p.Name != q.Name || len(p.Bounds) != len(q.Bounds) || p.Declaration == nil || p.Declaration.SourceName != q.Name {
			return false
		}
		for j, bound := range p.Bounds {
			if bound.Original != q.Bounds[j].Original || !helpfulSameBindings(bound.TypeParameterBindings, q.Bounds[j].TypeParameterBindings) {
				return false
			}
		}
	}
	return true
}

func lateSourceDefinition(definition *symbol.Definition) lateSourceDefinitionAnchor {
	return lateSourceDefinitionAnchor{definition: definition, name: definition.OriginalName, typ: definition.OriginalType, direct: definition.DirectTypeParameter, bindings: lateSourceCopyBindings(definition.TypeParameterBindings)}
}

func (a lateSourceDefinitionAnchor) matches(definition *symbol.Definition) bool {
	return definition != nil && definition == a.definition && definition.OriginalName == a.name && definition.OriginalType == a.typ && definition.DirectTypeParameter == a.direct && helpfulSameBindings(definition.TypeParameterBindings, a.bindings)
}

func newLateSourceOwnershipFrame(file parsing.SourceFile, ctx Ctx) *lateSourceOwnershipFrame {
	f := &lateSourceOwnershipFrame{invocation: ctx.syntaxInvocation, graph: symbol.GlobalScope, file: file.Symbols, root: file.Ast, active: true, owners: make(map[*symbol.ClassScope]lateSourceOwnerAnchor), methods: make(map[*symbol.Definition]lateSourceMethodAnchor), binders: make(map[*symbol.TypeParamDeclaration]lateSourceBinderAnchor), late: make(map[*symbol.ClassScope]lateSourceRegistration)}
	if f.file == nil || ctx.currentFile != f.file || f.root == nil || f.file.ParsedRoot() != f.root || f.graph == nil {
		f.active = false
		return f
	}
	if f.invocation != nil {
		f.session = f.invocation.session
	}
	count := 0
	for packageName, pkg := range f.graph.Packages {
		if pkg == nil {
			continue
		}
		for key, registered := range pkg.Files {
			if registered == f.file {
				count++
				f.packageName, f.packageScope, f.fileKey = packageName, pkg, key
			}
		}
	}
	if count != 1 || f.packageName != f.file.Package {
		f.close()
		return f
	}
	f.snapshot = f.file.SourceInputSnapshot()
	if !f.current() {
		return f
	}
	seen := map[*symbol.ClassScope]bool{}
	var visit func(*symbol.ClassScope)
	visit = func(owner *symbol.ClassScope) {
		if owner == nil || seen[owner] {
			return
		}
		seen[owner] = true
		if declaredRefusalOrdinaryScopeFile(owner) == f.file {
			f.recordOwner(owner)
		}
		for _, child := range owner.Subclasses {
			visit(child)
		}
	}
	for _, owner := range f.file.TopLevelClasses {
		visit(owner)
	}
	return f
}

func (f *lateSourceOwnershipFrame) close() {
	if f != nil {
		f.active = false
	}
}

// Only stored witness corruption retires a frame. Foreign queries are rejected
// before this check and cannot poison a genuine outer frame or another caller.
func (f *lateSourceOwnershipFrame) current() bool {
	if f == nil || !f.active {
		return false
	}
	if f.graph != symbol.GlobalScope || f.file == nil || f.root == nil || f.file.ParsedRoot() != f.root || f.root.HasChanges() || f.snapshot == nil || f.file.SourceInputSnapshot() != f.snapshot || f.snapshot.Validate() != nil || f.file.Package != f.packageName || f.graph.Packages[f.packageName] != f.packageScope || f.packageScope == nil || f.packageScope.Files[f.fileKey] != f.file {
		f.close()
		return false
	}
	found := 0
	for packageName, pkg := range f.graph.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == f.file {
				if packageName != f.file.Package {
					f.close()
					return false
				}
				found++
			}
		}
	}
	if found != 1 {
		f.close()
		return false
	}
	return true
}

func (f *lateSourceOwnershipFrame) recordBinders(parameters []symbol.TypeParam, owner *sitter.Node, class *symbol.ClassScope, method *symbol.Definition) bool {
	written := symbol.ExtractTypeParameters(owner.ChildByFieldName("type_parameters"), f.file.Source)
	nodes := nodeutil.SemanticNamedChildrenOf(owner.ChildByFieldName("type_parameters"))
	if len(parameters) != len(written) || len(nodes) != len(written) {
		return false
	}
	for i, parameter := range parameters {
		if parameter.Declaration == nil || parameter.Name != written[i].Name || parameter.Declaration.SourceName != written[i].Name || len(parameter.Bounds) != len(written[i].Bounds) {
			return false
		}
		for j, bound := range parameter.Bounds {
			if bound.Original != written[i].Bounds[j].Original {
				return false
			}
		}
		if old, exists := f.binders[parameter.Declaration]; exists && (old.node != nodes[i] || old.owner != owner) {
			return false
		}
	}
	for i, parameter := range parameters {
		f.binders[parameter.Declaration] = lateSourceBinderAnchor{node: nodes[i], owner: owner, class: class, method: method, parameter: lateSourceCopyParameters([]symbol.TypeParam{parameter})[0]}
	}
	return true
}

func (f *lateSourceOwnershipFrame) recordOwner(owner *symbol.ClassScope) bool {
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil || !threadSourceNodeWithin(owner.Class.DeclarationNode, f.root) {
		return false
	}
	node := owner.Class.DeclarationNode
	if node.Type() != "object_creation_expression" {
		name := node.ChildByFieldName("name")
		if name == nil || name.Content(f.file.Source) != owner.Class.OriginalName {
			return false
		}
	}
	own := owner.OwnTypeParameters()
	if node.Type() == "object_creation_expression" {
		own = nil
	}
	if !f.recordBinders(own, node, owner, nil) {
		return false
	}
	for _, parameter := range owner.TypeParameters {
		anchor, ok := f.binders[parameter.Declaration]
		if !ok || !threadSourceNodeWithin(node, anchor.owner) || !lateSourceParametersMatch([]symbol.TypeParam{parameter}, []symbol.TypeParam{anchor.parameter}) {
			return false
		}
	}
	f.owners[owner] = lateSourceOwnerAnchor{class: owner.Class, javaName: owner.Class.OriginalName, inner: owner.IsInner, iface: owner.IsInterface, node: node, enclosing: owner.Enclosing, parameters: lateSourceCopyParameters(owner.TypeParameters), own: lateSourceCopyParameters(own)}
	for _, method := range owner.Methods {
		if method == nil || method.DeclarationNode == nil || !lateSourceExecutableOwned(owner, method) {
			continue
		}
		declaration := method.DeclarationNode
		name, typ := declaration.ChildByFieldName("name"), declaration.ChildByFieldName("type")
		constructor := declaration.Type() == "constructor_declaration"
		if name == nil || name.Content(f.file.Source) != method.OriginalName || method.Constructor != constructor || !constructor && (typ == nil || typ.Content(f.file.Source) != method.OriginalType) || constructor && method.OriginalType != "" || method.IsStatic != declarationHasModifier(declaration, "static") || method.IsPrivate != declarationHasModifier(declaration, "private") || method.HasBody != (declaration.ChildByFieldName("body") != nil) || !f.recordBinders(method.TypeParameters, declaration, owner, method) {
			continue
		}
		parameters := nodeutil.SemanticNamedChildrenOf(declaration.ChildByFieldName("parameters"))
		if len(parameters) != len(method.Parameters) {
			continue
		}
		a := lateSourceMethodAnchor{owner: owner, node: declaration, definition: lateSourceDefinition(method), binders: lateSourceCopyParameters(method.TypeParameters), static: method.IsStatic, private: method.IsPrivate, constructor: method.Constructor, body: method.HasBody}
		valid := true
		for i, parameter := range parameters {
			typ, name := nodeutil.JavaParameterNodes(parameter)
			actual := method.Parameters[i]
			if actual == nil || typ == nil || name == nil || typ.Content(f.file.Source) != actual.OriginalType || name.Content(f.file.Source) != actual.OriginalName {
				valid = false
				break
			}
			a.parameters = append(a.parameters, lateSourceDefinition(actual))
		}
		if valid {
			f.methods[method] = a
		}
	}
	return true
}

// Called solely at actual synthesis/registration, before initializer or body
// emission. A caller-supplied registry alone cannot produce this witness.
func registerLateSourceAnonymous(info *anonymousClassInfo, key anonymousClassKey, ctx Ctx) {
	f := ctx.lateSourceOwnership
	if f == nil || f != activeLateSourceOwnership || !f.current() || ctx.currentFile != f.file || info == nil || ctx.anonymousClasses[key] != info {
		return
	}
	if f.anonymous == nil {
		f.anonymous = ctx.anonymousClasses
	}
	for owner, old := range f.late {
		if old.anonymous != nil && old.key == key {
			delete(f.late, owner)
		}
	}
	if f.recordOwner(info.scope) {
		if registration, ok := f.registration(info.scope, ctx); ok {
			registration.anonymous, registration.key = info, key
			f.late[info.scope] = registration
		}
	}
}

func registerLateSourceLocal(info *localClassInfo, name string, ctx Ctx) {
	f := ctx.lateSourceOwnership
	if f == nil || f != activeLateSourceOwnership || !f.current() || ctx.currentFile != f.file || info == nil || ctx.localClasses[name] != info {
		return
	}
	if f.locals == nil {
		f.locals = ctx.localClasses
	}
	for owner, old := range f.late {
		if old.local != nil && old.name == name {
			delete(f.late, owner)
		}
	}
	if f.recordOwner(info.scope) {
		if registration, ok := f.registration(info.scope, ctx); ok {
			registration.local, registration.name = info, name
			f.late[info.scope] = registration
		}
	}
}

func (f *lateSourceOwnershipFrame) ownerCurrent(owner *symbol.ClassScope) bool {
	a, exists := f.owners[owner]
	if !exists {
		return false
	}
	own := owner.OwnTypeParameters()
	if a.node.Type() == "object_creation_expression" {
		own = nil
	}
	if owner.Class != a.class || owner.Class.OriginalName != a.javaName || owner.IsInner != a.inner || owner.IsInterface != a.iface || owner.Class.DeclarationNode != a.node || owner.Enclosing != a.enclosing || !lateSourceParametersMatch(owner.TypeParameters, a.parameters) || !lateSourceParametersMatch(own, a.own) {
		f.close()
		return false
	}
	return true
}

func jdk21LateSourceDeclaringUseOwned(ctx Ctx) bool {
	f := ctx.lateSourceOwnership
	if f == nil || f != activeLateSourceOwnership || !f.active || ctx.currentFile != f.file || ctx.syntaxInvocation != f.invocation || ctx.currentClass == nil || ctx.localScope == nil {
		return false
	}
	registration, exists := f.late[ctx.currentClass]
	caller, methodExists := f.methods[ctx.localScope]
	if !exists || !methodExists || caller.owner != ctx.currentClass || f.callerOwner != ctx.currentClass || f.callerMethod != ctx.localScope {
		return false
	}
	if registration.anonymous != nil && (f.anonymous[registration.key] != registration.anonymous || registration.anonymous.scope != ctx.currentClass) || registration.local != nil && (f.locals[registration.name] != registration.local || registration.local.scope != ctx.currentClass) {
		f.close()
		return false
	}
	// Query identity is separate from the stored registry. A copied query map
	// with different info refuses without poisoning the actual stored witness.
	if registration.anonymous != nil && ctx.anonymousClasses[registration.key] != registration.anonymous || registration.local != nil && ctx.localClasses[registration.name] != registration.local {
		return false
	}
	if invocation := ctx.syntaxInvocation; invocation != nil {
		owned, present := invocation.files[f.file]
		if !invocation.active || invocation.session != f.session || !f.session.Active() || !standaloneSyntaxConfigurationOK(os.Stdout) || invocation.graph != f.graph || !present || owned.Symbols != f.file || owned.Ast != f.root || !bytes.Equal(owned.Source, f.file.Source) || activeResolutionFiles == nil || activeResolutionFiles.graph != f.graph || activeResolutionFiles.renderFile != f.file || activeResolutionFiles.declarations != invocation.declarations {
			return false
		}
	}
	if !f.current() {
		return false
	}
	seen := map[*symbol.ClassScope]bool{}
	child := ctx.currentClass
	for owner := ctx.currentClass; owner != nil; owner = owner.Enclosing {
		if seen[owner] || !f.ownerCurrent(owner) || !threadSourceNodeWithin(owner.Class.DeclarationNode, f.root) || child != owner && !threadSourceNodeWithin(child.Class.DeclarationNode, owner.Class.DeclarationNode) {
			return false
		}
		seen[owner] = true
		if registered, late := f.late[owner]; late {
			if !f.registrationCurrent(owner, registered) {
				f.close()
				return false
			}
		} else if declaredRefusalOrdinaryScopeFile(owner) != f.file {
			f.close()
			return false
		}
		child = owner
	}
	method := ctx.localScope
	if !f.methodCurrent(method) || !sourceReflectionDeclaredMethod(ctx.currentClass, method) {
		f.close()
		return false
	}
	for _, parameter := range append(append([]symbol.TypeParam(nil), ctx.currentClass.TypeParameters...), method.TypeParameters...) {
		if !f.binderCurrent(parameter) {
			f.close()
			return false
		}
	}
	return intrinsicLexicalDeclaringFile(ctx.currentClass, f.file, ctx)
}

// Binder identity is anchored at the trusted producer's actual source slot.
// GoName is intentionally live: hoisting can allocate a different Go spelling.
func (f *lateSourceOwnershipFrame) binderCurrent(parameter symbol.TypeParam) bool {
	a, present := f.binders[parameter.Declaration]
	if !present || !lateSourceParametersMatch([]symbol.TypeParam{parameter}, []symbol.TypeParam{a.parameter}) || a.node == nil || !threadSourceNodeWithin(a.node, f.root) || !f.ownerCurrent(a.class) {
		return false
	}
	if a.method == nil {
		for _, current := range a.class.OwnTypeParameters() {
			if current.Declaration == parameter.Declaration {
				return lateSourceParametersMatch([]symbol.TypeParam{current}, []symbol.TypeParam{a.parameter})
			}
		}
		return false
	}
	method, exists := f.methods[a.method]
	if !exists || method.node != a.owner || !f.methodCurrent(a.method) {
		return false
	}
	for _, current := range a.method.TypeParameters {
		if current.Declaration == parameter.Declaration {
			return lateSourceParametersMatch([]symbol.TypeParam{current}, []symbol.TypeParam{a.parameter})
		}
	}
	return false
}

// Actual method emission owns this stack entry. A method-table pointer alone
// cannot establish which of two genuine methods is currently being emitted.
func enterLateSourceMethod(ctx Ctx) func() {
	f := ctx.lateSourceOwnership
	if f == nil || f != activeLateSourceOwnership || !f.active {
		return func() {}
	}
	method, present := f.methods[ctx.localScope]
	if !present || method.owner != ctx.currentClass {
		return func() {}
	}
	oldOwner, oldMethod := f.callerOwner, f.callerMethod
	f.callerOwner, f.callerMethod = ctx.currentClass, ctx.localScope
	return func() { f.callerOwner, f.callerMethod = oldOwner, oldMethod }
}

// Captured binders may originate in an enclosing generic constructor as well
// as a method. This records that actual grammar role without treating a
// constructor as an active anonymous method caller.
func lateSourceExecutableOwned(owner *symbol.ClassScope, method *symbol.Definition) bool {
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil || method == nil || method.DeclarationNode == nil {
		return false
	}
	if method.DeclarationNode.Type() != "method_declaration" && method.DeclarationNode.Type() != "constructor_declaration" {
		return false
	}
	for parent := method.DeclarationNode.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Equal(owner.Class.DeclarationNode) {
			return true
		}
		switch parent.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "object_creation_expression":
			return false
		}
	}
	return false
}

// Both a current late caller and a captured enclosing executable use the same
// semantic header check. Generated Go names and local body state stay live.
func (f *lateSourceOwnershipFrame) methodCurrent(method *symbol.Definition) bool {
	a, exists := f.methods[method]
	if !exists || !f.ownerCurrent(a.owner) || method.DeclarationNode != a.node || !lateSourceExecutableOwned(a.owner, method) || !a.definition.matches(method) || method.IsStatic != a.static || method.IsPrivate != a.private || method.Constructor != a.constructor || method.HasBody != a.body || !lateSourceParametersMatch(method.TypeParameters, a.binders) || len(method.Parameters) != len(a.parameters) {
		return false
	}
	member := false
	for _, actual := range a.owner.Methods {
		if actual == method {
			member = true
			break
		}
	}
	if !member {
		return false
	}
	for i, parameter := range a.parameters {
		if !parameter.matches(method.Parameters[i]) {
			return false
		}
	}
	return true
}

// The producer's executable comes from the actual source ancestry, not a
// retained localScope label. Field/initializer roles carry no fabricated
// executable. A lambda may have a synthetic local scope, but its real enclosing
// Java executable still has an ordinary source-owned header anchor.
func (f *lateSourceOwnershipFrame) registration(scope *symbol.ClassScope, ctx Ctx) (lateSourceRegistration, bool) {
	r := lateSourceRegistration{producer: ctx.currentClass}
	parent, exists := f.owners[r.producer]
	if !exists || scope.Enclosing != r.producer {
		return r, false
	}
	lambda := false
	var initializer *sitter.Node
	for node := scope.Class.DeclarationNode.Parent(); node != nil && !node.Equal(parent.node); node = node.Parent() {
		switch node.Type() {
		case "lambda_expression":
			lambda = true
		case "field_declaration", "static_initializer":
			initializer = node
		case "block":
			if node.Parent() != nil && node.Parent().Type() == "class_body" {
				initializer = node
			}
		case "method_declaration", "constructor_declaration":
			for method, a := range f.methods {
				if a.owner == r.producer && a.node == node {
					if !f.methodCurrent(method) {
						return r, false
					}
					if ctx.localScope == nil || ctx.localScope.DeclarationNode != nil && ctx.localScope != method || ctx.localScope.DeclarationNode == nil && !lambda {
						return r, false
					}
					r.executable, r.role = method, node
					return r, true
				}
			}
			return r, false
		case "object_creation_expression":
			// Constructor arguments are evaluated in the enclosing lexical
			// scope; only the actual anonymous body introduces an owner.
			if body := objectCreationClassBody(node); body != nil && threadSourceNodeWithin(scope.Class.DeclarationNode, body) {
				return r, false
			}
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			return r, false
		}
	}
	if initializer != nil {
		r.role = initializer
		return r, true
	}
	return r, false
}

func (f *lateSourceOwnershipFrame) registrationCurrent(scope *symbol.ClassScope, r lateSourceRegistration) bool {
	if r.producer == nil || scope.Enclosing != r.producer || !f.ownerCurrent(r.producer) || r.role == nil || !threadSourceNodeWithin(scope.Class.DeclarationNode, r.role) {
		return false
	}
	if r.anonymous != nil && (f.anonymous[r.key] != r.anonymous || r.anonymous.scope != scope) || r.local != nil && (f.locals[r.name] != r.local || r.local.scope != scope) {
		return false
	}
	if r.executable != nil {
		a, exists := f.methods[r.executable]
		return exists && a.owner == r.producer && a.node == r.role && f.methodCurrent(r.executable)
	}
	switch r.role.Type() {
	case "field_declaration", "static_initializer", "block":
		return threadSourceNodeWithin(r.role, r.producer.Class.DeclarationNode)
	}
	return false
}
