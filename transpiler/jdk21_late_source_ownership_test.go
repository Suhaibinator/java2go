package transpiler

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const lateSourceOwnershipFieldJava = `import java.time.LocalDate;
interface DateView { LocalDate read(LocalDate value); }
class Owner {
  static DateView field = new DateView() {
    Object marker;
    public LocalDate read(LocalDate value) { return value; }
  };
}`

const lateSourceOwnershipLambdaJava = `import java.time.LocalDate;
interface DateView { LocalDate read(LocalDate value); }
interface Factory { DateView make(); }
class Owner {
  DateView create() {
    Factory factory = () -> new DateView() {
      Object marker;
      public LocalDate read(LocalDate value) { return value; }
    };
    return factory.make();
  }
}`

// Every identity queried below originates in actual parsing, symbol resolution
// and production anonymous/local lowering. No selected method or inferred
// argument/carrier is supplied to make an overload succeed.
func lateSourceOwnershipFixture(t *testing.T, source string) (Ctx, *lateSourceOwnershipFrame, parsing.SourceFile) {
	t.Helper()
	return lateSourceOwnershipFixtureMode(t, source, false)
}

func lateSourceOwnershipFixtureMode(t *testing.T, source string, ownedSession bool) (Ctx, *lateSourceOwnershipFrame, parsing.SourceFile) {
	t.Helper()
	oldGraph, oldIndex, oldImports, oldFrame := symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership = nil, nil, nil
	t.Cleanup(func() {
		symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership = oldGraph, oldIndex, oldImports, oldFrame
		setStrictMode(false)
		resetDiagnostics()
	})
	file := parsing.SourceFile{Name: "LateOwned.java", Source: []byte(source)}
	if err := file.ParseAST(); err != nil || file.Ast == nil || file.Ast.HasError() {
		t.Fatal("actual source parse failed", err)
	}
	symbol.AddSymbolsToPackage(file.ParseSymbols())
	ResolveFiles([]parsing.SourceFile{file})
	var hoisted []ast.Decl
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, genericFamilies: &genericFamilyAnalysis{}, callableSubclasses: &callableSubclassSourceInventory{}, runtimeLocationProfile: sourceRuntimeLocationClasspath, anonymousClasses: make(map[anonymousClassKey]*anonymousClassInfo), localClasses: make(map[string]*localClassInfo), hoistedDecls: &hoisted, anonClassCounter: new(int)}
	if ownedSession {
		session := cliownership.New()
		t.Cleanup(session.Close)
		ctx.syntaxInvocation = newSyntaxInvocation(session, []parsing.SourceFile{file})
		if ctx.syntaxInvocation == nil {
			t.Fatal("actual source session unavailable")
		}
		t.Cleanup(ctx.syntaxInvocation.close)
	}
	activeResolutionFiles = sourceOwnershipIndex(ctx)
	activeImportSourceInventory = resolvedSourceInventory(ctx)
	frame := newLateSourceOwnershipFrame(file, ctx)
	ctx.lateSourceOwnership, activeLateSourceOwnership = frame, frame
	t.Cleanup(frame.close)
	if !frame.active {
		t.Fatal("actual resolved source frame unavailable")
	}
	ParseNode(file.Ast, file.Source, ctx)
	for owner := range frame.late {
		for _, method := range owner.Methods {
			if method != nil && method.OriginalName == "read" {
				ctx.currentClass, ctx.localScope = owner, method
				outerLocalClasses := ctx.localClasses
				registration := frame.late[owner]
				if registration.local != nil {
					// A Java block owns its copied local-name map. The root
					// context deliberately never receives that declaration.
					// Recover only the exact map retained by the real producer;
					// never allocate info, scopes, headers, binders or witnesses.
					name := owner.Class.DeclarationNode.ChildByFieldName("name")
					if name == nil || name.Content(file.Source) != registration.name || registration.local.scope != owner || frame.locals == nil || frame.locals[registration.name] != registration.local || outerLocalClasses[registration.name] == registration.local {
						t.Fatal("actual block-local producer registry identity unavailable")
					}
					ctx.localClasses = frame.locals
				}
				t.Cleanup(enterLateSourceMethod(ctx))
				if registration.local != nil {
					outside := ctx.Clone()
					outside.localClasses = outerLocalClasses
					if jdk21LateSourceDeclaringUseOwned(outside) || !frame.active {
						t.Fatal("outer block acquired a local binding or poisoned its genuine caller")
					}
				}
				return ctx, frame, file
			}
		}
	}
	t.Fatal("production lowering did not register the source read caller")
	return Ctx{}, nil, parsing.SourceFile{}
}

func TestJDK21LateSourceOwnershipProductionAndCopies(t *testing.T) {
	ctx, frame, _ := lateSourceOwnershipFixture(t, lateSourceOwnershipGenericJava)
	if findFileScopeForClassScope(ctx.currentClass, ctx) != nil {
		t.Fatal("fixture must be a genuine late source scope")
	}
	if len(ctx.currentClass.TypeParameters) < 2 {
		t.Fatal("actual class and method captures were not carried")
	}
	for _, copy := range []Ctx{ctx, ctx.Clone()} {
		if !jdk21LateSourceDeclaringUseOwned(copy) || !jdk21NominalDeclaringUseOwned(copy, "LocalDate") {
			t.Fatal("genuine generic source caller/value copy refused")
		}
	}
	for i := 0; i < 3; i++ {
		if !jdk21LateSourceDeclaringUseOwned(ctx) {
			t.Fatal("unchanged query lost source ownership")
		}
	}
	if !frame.active {
		t.Fatal("legitimate copies retired their frame")
	}
	declaration := ctx.currentClass.TypeParameters[0].Declaration
	oldName := declaration.GoName
	declaration.GoName = oldName + "_renamed"
	if !jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("legitimate generated binder spelling evolution changed Java identity")
	}
	declaration.GoName = oldName
	for _, entry := range []struct {
		name, source string
		executable   bool
	}{{"field_role", lateSourceOwnershipFieldJava, false}, {"lambda_role", lateSourceOwnershipLambdaJava, true}} {
		t.Run(entry.name, func(t *testing.T) {
			actual, frame, _ := lateSourceOwnershipFixture(t, entry.source)
			registration := frame.late[actual.currentClass]
			if (registration.executable != nil) != entry.executable || registration.role == nil || !jdk21LateSourceDeclaringUseOwned(actual) {
				t.Fatal("genuine source initializer/lambda role refused or fabricated executable")
			}
		})
	}
}

func TestJDK21LateSourceOwnershipDonationDoesNotPoison(t *testing.T) {
	ctx, frame, _ := lateSourceOwnershipFixture(t, lateSourceOwnershipGenericJava)
	assertForeign := func(name string, foreign Ctx) {
		t.Helper()
		if jdk21LateSourceDeclaringUseOwned(foreign) {
			t.Fatal("foreign authority accepted", name)
		}
		if !frame.active || !jdk21LateSourceDeclaringUseOwned(ctx) {
			t.Fatal("foreign query poisoned genuine render", name)
		}
	}
	fileCopy := *ctx.currentFile
	foreign := ctx.Clone()
	foreign.currentFile = &fileCopy
	assertForeign("file", foreign)
	ownerCopy := *ctx.currentClass
	foreign = ctx.Clone()
	foreign.currentClass = &ownerCopy
	assertForeign("owner", foreign)
	methodCopy := *ctx.localScope
	foreign = ctx.Clone()
	foreign.localScope = &methodCopy
	assertForeign("caller", foreign)
	for _, method := range ctx.currentClass.Methods {
		if method != ctx.localScope && method.DeclarationNode != nil {
			foreign = ctx.Clone()
			foreign.localScope = method
			assertForeign("other genuine method outside active caller", foreign)
		}
	}
	for key, info := range ctx.anonymousClasses {
		if info.scope == ctx.currentClass {
			infoCopy := *info
			foreign = ctx.Clone()
			foreign.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{key: &infoCopy}
			assertForeign("copied registry info", foreign)
		}
	}
}

func TestJDK21LateSourceOwnershipSourceRetirement(t *testing.T) {
	for _, change := range []string{"source", "original", "filename", "graph", "same_graph_removed_class", "caller_header", "binder_pointer", "registry_removed", "enclosing_cycle", "caller_parameter", "class_identity", "offsets", "class_java_name", "class_inner_flag", "class_interface_flag", "creator_static", "creator_private", "creator_constructor", "creator_body", "creator_formal", "creator_class_only_static", "creator_no_binders_static"} {
		t.Run(change, func(t *testing.T) {
			source := lateSourceOwnershipGenericJava
			if change == "creator_class_only_static" {
				source = strings.Replace(source, "MethodValue second = captured;", "int second = 1;", 1)
			}
			if change == "creator_no_binders_static" {
				source = strings.Replace(source, "class Owner<ClassValue>", "class Owner", 1)
				source = strings.Replace(source, "<MethodValue> DateView create(ClassValue owner, MethodValue captured)", "DateView create(Object owner, Object captured)", 1)
				source = strings.Replace(source, "ClassValue first = owner;", "Object first = owner;", 1)
				source = strings.Replace(source, "MethodValue second = captured;", "Object second = captured;", 1)
			}
			ctx, frame, _ := lateSourceOwnershipFixture(t, source)
			if change == "creator_class_only_static" && len(ctx.currentClass.TypeParameters) != 1 {
				t.Fatal("class-only producer fixture must not carry method binder")
			}
			if change == "creator_no_binders_static" && len(ctx.currentClass.TypeParameters) != 0 {
				t.Fatal("binder-free producer fixture must carry no binder")
			}
			var restore func()
			switch change {
			case "source":
				old := ctx.currentFile.Source[0]
				ctx.currentFile.Source[0] = ' '
				restore = func() { ctx.currentFile.Source[0] = old }
			case "original":
				old := ctx.currentFile.OriginalSource
				ctx.currentFile.OriginalSource = []byte("changed")
				restore = func() { ctx.currentFile.OriginalSource = old }
			case "filename":
				old := ctx.currentFile.SourceFilename
				ctx.currentFile.SourceFilename = "Other.java"
				restore = func() { ctx.currentFile.SourceFilename = old }
			case "graph":
				old := symbol.GlobalScope
				symbol.GlobalScope = &symbol.GlobalSymbols{Packages: old.Packages}
				restore = func() { symbol.GlobalScope = old }
			case "same_graph_removed_class":
				old := ctx.currentFile.TopLevelClasses
				ctx.currentFile.TopLevelClasses = nil
				restore = func() { ctx.currentFile.TopLevelClasses = old }
			case "caller_header":
				old := ctx.localScope.OriginalName
				ctx.localScope.OriginalName = "donated"
				restore = func() { ctx.localScope.OriginalName = old }
			case "class_java_name":
				old := ctx.currentClass.Class.OriginalName
				ctx.currentClass.Class.OriginalName = "Donated"
				restore = func() { ctx.currentClass.Class.OriginalName = old }
			case "class_inner_flag":
				old := ctx.currentClass.IsInner
				ctx.currentClass.IsInner = !old
				restore = func() { ctx.currentClass.IsInner = old }
			case "class_interface_flag":
				old := ctx.currentClass.IsInterface
				ctx.currentClass.IsInterface = !old
				restore = func() { ctx.currentClass.IsInterface = old }
			case "creator_static", "creator_class_only_static", "creator_no_binders_static":
				creator := frame.late[ctx.currentClass].executable
				if creator == nil {
					t.Fatal("actual source creator method anchor missing")
				}
				old := creator.IsStatic
				creator.IsStatic = !old
				restore = func() { creator.IsStatic = old }
			case "creator_private":
				creator := frame.late[ctx.currentClass].executable
				old := creator.IsPrivate
				creator.IsPrivate = !old
				restore = func() { creator.IsPrivate = old }
			case "creator_constructor":
				creator := frame.late[ctx.currentClass].executable
				old := creator.Constructor
				creator.Constructor = !old
				restore = func() { creator.Constructor = old }
			case "creator_body":
				creator := frame.late[ctx.currentClass].executable
				old := creator.HasBody
				creator.HasBody = !old
				restore = func() { creator.HasBody = old }
			case "creator_formal":
				creator := frame.late[ctx.currentClass].executable
				old := creator.Parameters[0].OriginalType
				creator.Parameters[0].OriginalType = "Object"
				restore = func() { creator.Parameters[0].OriginalType = old }
			case "enclosing_cycle":
				old := ctx.currentClass.Enclosing
				ctx.currentClass.Enclosing = ctx.currentClass
				restore = func() { ctx.currentClass.Enclosing = old }
			case "caller_parameter":
				old := ctx.localScope.Parameters[0].OriginalType
				ctx.localScope.Parameters[0].OriginalType = "Object"
				restore = func() { ctx.localScope.Parameters[0].OriginalType = old }
			case "class_identity":
				old := ctx.currentClass.Class
				copy := *old
				ctx.currentClass.Class = &copy
				restore = func() { ctx.currentClass.Class = old }
			case "offsets":
				old := ctx.currentFile.SourceOffsets
				ctx.currentFile.SourceOffsets = []uint32{0}
				restore = func() { ctx.currentFile.SourceOffsets = old }
			case "registry_removed":
				for key, info := range ctx.anonymousClasses {
					if info.scope == ctx.currentClass {
						delete(ctx.anonymousClasses, key)
						restore = func() { ctx.anonymousClasses[key] = info }
						break
					}
				}
			case "binder_pointer":
				old := ctx.currentClass.TypeParameters[0].Declaration
				copy := *old
				ctx.currentClass.TypeParameters[0].Declaration = &copy
				restore = func() { ctx.currentClass.TypeParameters[0].Declaration = old }
			}
			if jdk21LateSourceDeclaringUseOwned(ctx) || frame.active {
				t.Fatal("actual stored witness mutation was not retired", change)
			}
			restore()
			if jdk21LateSourceDeclaringUseOwned(ctx) {
				t.Fatal("restored data revived retired frame")
			}
		})
	}
}

func TestJDK21LateSourceOwnershipFrameLifetime(t *testing.T) {
	ctx, outer, file := lateSourceOwnershipFixture(t, lateSourceOwnershipGenericJava)
	inner := newLateSourceOwnershipFrame(file, ctx)
	activeLateSourceOwnership = inner
	if jdk21LateSourceDeclaringUseOwned(ctx) || !outer.active {
		t.Fatal("outer context crossed nested frame or was poisoned")
	}
	inner.close()
	activeLateSourceOwnership = outer
	if !jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("actual outer stack restoration failed fresh ownership")
	}
	base := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, callableSubclasses: &callableSubclassSourceInventory{}, genericFamilies: &genericFamilyAnalysis{}, runtimeLocationProfile: sourceRuntimeLocationClasspath}
	if _, err := convertFileNode(file, base); err != nil {
		t.Fatal("actual same-file nested production render failed", err)
	}
	if activeLateSourceOwnership != outer || !outer.active || !jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("actual production nested render did not restore genuine outer frame")
	}
	outer.close()
	if jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("closed frame reused")
	}
}

func TestJDK21LateSourceOwnershipLocalGenericProduction(t *testing.T) {
	ctx, frame, _ := lateSourceOwnershipFixture(t, `import java.time.LocalDate;
class Owner<ClassValue> {
  <MethodValue> Object create(ClassValue owner, MethodValue captured) {
    class Local<OwnValue> {
      ClassValue first = owner; MethodValue second = captured; OwnValue third;
      Local(OwnValue value) { third = value; }
      LocalDate read(LocalDate value) { return value; }
    }
    return new Local<String>("value");
  }
}`)
	if len(ctx.currentClass.TypeParameters) < 3 || !jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("genuine local own/class/method binders refused")
	}
	registration := frame.late[ctx.currentClass]
	if registration.executable == nil || registration.producer != ctx.currentClass.Enclosing {
		t.Fatal("genuine enclosing executable/class identity missing")
	}
	for _, source := range []struct {
		name       string
		parameters []symbol.TypeParam
		owner      *symbol.ClassScope
		method     *symbol.Definition
	}{
		{"ClassValue", registration.producer.OwnTypeParameters(), registration.producer, nil},
		{"MethodValue", registration.executable.TypeParameters, registration.producer, registration.executable},
		{"OwnValue", ctx.currentClass.OwnTypeParameters(), ctx.currentClass, nil},
	} {
		var actual *symbol.TypeParamDeclaration
		for _, parameter := range source.parameters {
			if parameter.Name == source.name {
				if actual != nil {
					t.Fatal("duplicate source binder slot", source.name)
				}
				actual = parameter.Declaration
			}
		}
		anchor, present := frame.binders[actual]
		if actual == nil || actual.SourceName != source.name || visibleTypeParameterDeclarationForJavaType(source.name, ctx) != actual || !present || anchor.class != source.owner || anchor.method != source.method || anchor.node == nil || !threadSourceNodeWithin(anchor.node, frame.root) {
			t.Fatal("actual local source binder origin/visibility identity disagrees", source.name)
		}
	}
}

func TestJDK21LateSourceOwnershipOptionalSession(t *testing.T) {
	ctx, frame, file := lateSourceOwnershipFixtureMode(t, lateSourceOwnershipGenericJava, true)
	invocation := ctx.syntaxInvocation
	if invocation.declarations == nil || !invocation.declarations.retired {
		t.Fatal("fixture must exercise independently retired256-byte optimization")
	}
	if !jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("live session must not depend on optional named declaration budget")
	}
	wrong := ctx.Clone()
	wrong.syntaxInvocation = newSyntaxInvocation(invocation.session, []parsing.SourceFile{file})
	if wrong.syntaxInvocation == nil {
		t.Fatal("second actual capability unavailable")
	}
	t.Cleanup(wrong.syntaxInvocation.close)
	if jdk21LateSourceDeclaringUseOwned(wrong) || !frame.active {
		t.Fatal("wrong supplied capability borrowed holder or poisoned frame")
	}
	originalSession := invocation.session
	originalSession.Close()
	donatedSession := cliownership.New()
	t.Cleanup(donatedSession.Close)
	invocation.session = donatedSession
	if jdk21LateSourceDeclaringUseOwned(ctx) || !frame.active {
		t.Fatal("donated active session revived original closed authority or poisoned source frame")
	}
	invocation.session = originalSession
	if jdk21LateSourceDeclaringUseOwned(ctx) {
		t.Fatal("closed original session remained eligible")
	}
	invocation.close()
	if jdk21LateSourceDeclaringUseOwned(ctx) || !frame.active {
		t.Fatal("closed supplied capability fell back or poisoned frame")
	}
	missing := ctx.Clone()
	missing.syntaxInvocation = nil
	if jdk21LateSourceDeclaringUseOwned(missing) || !frame.active {
		t.Fatal("dropping supplied closed capability acquired legacy permission")
	}
	t.Run("separate_actual_legacy_render", func(t *testing.T) {
		legacy, _, _ := lateSourceOwnershipFixture(t, lateSourceOwnershipGenericJava)
		if !jdk21LateSourceDeclaringUseOwned(legacy) {
			t.Fatal("valid source-owned nil-capability legacy frame refused")
		}
	})
}

func TestJDK21LateSourceOwnershipNamespaceShields(t *testing.T) {
	for _, binder := range []string{"ClassValue", "MethodValue"} {
		t.Run(binder, func(t *testing.T) {
			source := strings.Replace(lateSourceOwnershipGenericJava,
				"interface DateView { LocalDate read(LocalDate value); }",
				"interface DateView<DateType> { DateType read(DateType value); }", 1)
			source = strings.Replace(source, "<MethodValue> DateView create(", "<MethodValue> DateView<LocalDate> create(", 1)
			source = strings.Replace(source, "new DateView()", "new DateView<LocalDate>()", 1)
			source = strings.ReplaceAll(source, binder, "LocalDate")
			ctx, _, _ := lateSourceOwnershipFixture(t, source)
			if !jdk21LateSourceDeclaringUseOwned(ctx) {
				t.Fatal("source ownership itself should remain genuine")
			}
			if jdk21NominalDeclaringUseOwned(ctx, "LocalDate") || declaredExternalNominalOwner("LocalDate", nil, ctx) != "" {
				t.Fatal("actual source binder homonym acquired JDK identity")
			}
		})
	}
}
