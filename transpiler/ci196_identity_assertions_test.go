package transpiler

import (
	"github.com/NickyBoy89/java2go/stdjava"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

const ci196QualifiedEnumOriginalJava = `
public class E {
    enum Day { MON, TUE, WED }
    public Day pick() {
        return Day.WED;
    }
    public boolean isWed(Day d) {
        return d == Day.WED;
    }
}
`

const ci196GapFeaturesOriginalJava = `
import java.util.List;
import java.util.stream.Collectors;

public class GapFeatures<T extends Number> {
    private final Class<T> type;
    private double x, y;

    public GapFeatures(Class<T> type, double x, double y) {
        this.type = type;
        this.x = x;
        this.y = y;
    }

    public List<Double> process(List<T> inputs) {
        return inputs.stream()
            .filter(n -> n.doubleValue() > 0)
            .map(n -> Math.sqrt(n.doubleValue()) * Math.PI)
            .collect(Collectors.toList());
    }

    public boolean isDouble() {
        return type == Double.class;
    }

    public double coordinates() {
        return x + y;
    }
}
`

const ci196QualifiedEnumOriginalGoChild = `package main

import (
    "slices"
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func TestQualifiedEnumOriginal196Behavior(t *testing.T) {
    // Exercise the original emitted file; a zero-valued E is its empty default instance.
    subject := &E{}
    execution := stdjava.NewExecution()
    if MON != nil || TUE != nil || WED != nil {
        t.Fatal("qualified enum constants initialized before the first source access")
    }
    if subject.IsWedJava2goExecution(execution, nil) {
        t.Fatal("null compared equal to WED")
    }
    if MON == nil || TUE == nil || WED == nil {
        t.Fatal("a false null comparison failed to initialize its RHS enum constant")
    }
    original := WED
    if got := subject.PickJava2goExecution(execution); got != original {
        t.Fatal("pick did not return the initialized WED singleton")
    }
    if got := subject.PickJava2goExecution(execution); got != original || WED != original {
        t.Fatal("repeated source access changed enum singleton identity")
    }
    if !subject.IsWedJava2goExecution(execution, WED) || subject.IsWedJava2goExecution(execution, MON) || subject.IsWedJava2goExecution(execution, TUE) || subject.IsWedJava2goExecution(execution, nil) {
        t.Fatal("qualified enum comparison lost Java reference identity")
    }
    if WED.Ordinal() != 2 || !slices.Equal(WED.Name().UTF16Copy(), []uint16{87, 69, 68}) {
        t.Fatal("WED metadata changed")
    }
}
`

const ci196GapFeaturesOriginalGoChild = `
package main

import (
    "math"
    "testing"

    "github.com/NickyBoy89/java2go/stdjava"
)

func TestGapFeaturesRuntime(t *testing.T) {
    program := NewGapFeatures[*stdjava.Double](
        stdjava.ClassLiteral(stdjava.DoubleTypeID),
        1.25,
        2.5,
    )
    if !program.IsDouble() {
        t.Fatal("Double.class identity comparison was false")
    }
    if got := program.Coordinates(); got != 3.75 {
        t.Fatalf("Coordinates() = %v, want 3.75", got)
    }
    got := program.Process(stdjava.NewListFrom[*stdjava.Double](stdjava.BoxDouble(-1), stdjava.BoxDouble(4)))
    values := got.Slice()
    if len(values) != 1 || values[0] == nil || values[0].DoubleValue() != 2*math.Pi {
        t.Fatalf("Process() = %v, want [%v]", values, 2*math.Pi)
    }
}
`

func TestCI196OriginalQualifiedEnumBehavior(t *testing.T) {
	runGoTestInTempModule(t, renderGoFileFromJava(t, ci196QualifiedEnumOriginalJava), ci196QualifiedEnumOriginalGoChild)
}

func TestCI196OriginalGapFeaturesBehavior(t *testing.T) {
	runGoTestInTempModule(t, renderGoFileFromJava(t, ci196GapFeaturesOriginalJava), ci196GapFeaturesOriginalGoChild)
}

type ci196IdentityParameter struct{ name, typ string }

func ci196GuardedIdentityReturn(t ciABI193Reporter, file *ast.File, name, receiver, receiverType string, parameters []ci196IdentityParameter) (*ast.FuncDecl, ast.Expr) {
	t.Helper()
	runtimeImports := 0
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("runtime import is invalid: %v", err)
		}
		if path == "github.com/NickyBoy89/java2go/stdjava" {
			if spec.Name != nil && spec.Name.Name != "stdjava" {
				t.Fatalf("runtime import changed its generated binding")
			}
			runtimeImports++
		} else if spec.Name != nil && spec.Name.Name == "stdjava" {
			t.Fatalf("runtime import binding points to %s", path)
		}
	}
	if runtimeImports != 1 {
		t.Fatalf("runtime import count = %d, want one", runtimeImports)
	}
	method := ciABI193Function(t, file, name)
	if method.Recv == nil || len(method.Recv.List) != 1 || len(method.Recv.List[0].Names) != 1 || method.Recv.List[0].Names[0].Name != receiver {
		t.Fatalf("%s changed receiver identity", name)
	}
	ciABI193RequireExpr(t, method.Recv.List[0].Type, receiverType)
	if method.Type.Params == nil || len(method.Type.Params.List) != len(parameters) {
		t.Fatalf("%s changed parameter count", name)
	}
	for i, parameter := range parameters {
		field := method.Type.Params.List[i]
		if len(field.Names) != 1 || field.Names[0].Name != parameter.name {
			t.Fatalf("%s changed parameter %d identity", name, i)
		}
		ciABI193RequireExpr(t, field.Type, parameter.typ)
	}
	if method.Type.Results == nil || len(method.Type.Results.List) != 1 {
		t.Fatalf("%s changed result count", name)
	}
	ciABI193RequireExpr(t, method.Type.Results.List[0].Type, "bool")
	if method.Body == nil || len(method.Body.List) != 2 {
		t.Fatalf("%s must retain its receiver guard and one return", name)
	}
	guard, ok := method.Body.List[0].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || guard.Body == nil || len(guard.Body.List) != 1 {
		t.Fatalf("%s lost its receiver guard", name)
	}
	ciABI193RequireExpr(t, guard.Cond, receiver+" == nil")
	dereference, ok := guard.Body.List[0].(*ast.AssignStmt)
	if !ok || dereference.Tok != token.ASSIGN || len(dereference.Lhs) != 1 || len(dereference.Rhs) != 1 {
		t.Fatalf("%s lost its receiver guard dereference", name)
	}
	ciABI193RequireExpr(t, dereference.Lhs[0], "_")
	ciABI193RequireExpr(t, dereference.Rhs[0], "*"+receiver)
	result, ok := method.Body.List[1].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatalf("%s must have one identity return", name)
	}
	return method, result.Results[0]
}

func ci196AssertQualifiedEnumIdentity(t ciABI193Reporter, source string) {
	t.Helper()
	file := ciABI193Parse(t, source)
	method, result := ci196GuardedIdentityReturn(t, file, "IsWedJava2goExecution", "ee", "*E", []ci196IdentityParameter{{"__java2goExecution", "*stdjava.Execution"}, {"d", "*Eday"}})
	ciABI193RequireExpr(t, result, `stdjava.JavaReferenceEqual(stdjava.EvaluationValue(d), func() *Eday { EdayJava2goEnsureInitialized(__java2goExecution); return WED }())`)
	ciABI193RequireCallCount(t, method, "stdjava.EvaluationValue(d)", 1)
	ciABI193RequireCallCount(t, method, "EdayJava2goEnsureInitialized(__java2goExecution)", 1)
}

func ci196AssertDoubleClassIdentity(t ciABI193Reporter, source string) {
	t.Helper()
	file := ciABI193Parse(t, source)
	ciABI193RequireExpr(t, ciABI193Field(t, file, "GapFeatures", "type0"), "*stdjava.Class")
	_, result := ci196GuardedIdentityReturn(t, file, "IsDoubleJava2goExecution", "gs", "*GapFeatures[T]", []ci196IdentityParameter{{"__java2goExecution", "*stdjava.Execution"}})
	identity, ok := result.(*ast.BinaryExpr)
	if !ok || identity.Op != token.EQL {
		t.Fatalf("class identity comparison changed")
	}
	ciABI193RequireExpr(t, identity.X, "gs.type0")
	literal, ok := identity.Y.(*ast.CallExpr)
	if !ok || len(literal.Args) != 1 || literal.Ellipsis.IsValid() {
		t.Fatalf("class identity lost its ClassLiteral call")
	}
	ciABI193RequireExpr(t, literal.Fun, "stdjava.ClassLiteral")
	descriptor := ciABI193Text(t, literal.Args[0])
	typed := "stdjava.TypeID(" + strconv.Quote(string(stdjava.DoubleTypeID)) + ")"
	if descriptor != "stdjava.DoubleTypeID" && descriptor != typed {
		t.Fatalf("class literal descriptor = %s, want canonical java.lang.Double", descriptor)
	}
}

const ci196QualifiedIdentityGuardSource = `package p
import stdjava "github.com/NickyBoy89/java2go/stdjava"
func (ee *E) IsWedJava2goExecution(__java2goExecution *stdjava.Execution, d *Eday) bool {
    if ee == nil { _ = *ee }
    return stdjava.JavaReferenceEqual(stdjava.EvaluationValue(d), func() *Eday { EdayJava2goEnsureInitialized(__java2goExecution); return WED }())
}`

const ci196DoubleClassIdentityGuardSource = `package p
import stdjava "github.com/NickyBoy89/java2go/stdjava"
type GapFeatures[T stdjava.JavaNumber] struct { type0 *stdjava.Class }
func (gs *GapFeatures[T]) IsDoubleJava2goExecution(__java2goExecution *stdjava.Execution) bool {
    if gs == nil { _ = *gs }
    return gs.type0 == stdjava.ClassLiteral(stdjava.TypeID("java.lang.Double"))
}`

func ci196MutateIdentityGuard(t *testing.T, source, old, replacement string) string {
	t.Helper()
	if !strings.Contains(source, old) || old == replacement {
		t.Fatalf("invalid identity counter premise %q", old)
	}
	return strings.Replace(source, old, replacement, 1)
}

func TestCI196QualifiedEnumASTGuards(t *testing.T) {
	ci196AssertQualifiedEnumIdentity(t, ci196QualifiedIdentityGuardSource)
	for _, row := range []struct{ name, old, replacement, message string }{
		{"method", "IsWedJava2goExecution", "OtherJava2goExecution", "missing generated function"},
		{"receiver", "(ee *E)", "(ee *Other)", "generated expression"},
		{"enumFormal", "d *Eday", "d *Other", "generated expression"},
		{"parameterIdentity", "d *Eday", "value *Eday", "parameter"},
		{"executionFormal", "__java2goExecution *stdjava.Execution", "__java2goExecution any", "generated expression"},
		{"receiverGuard", "if ee == nil { _ = *ee }", "if ee == nil {}", "receiver guard"},
		{"guardOperand", "ee == nil", "d == nil", "generated expression"},
		{"wrongEquality", "stdjava.JavaReferenceEqual", "stdjava.ObjectEquals", "generated expression"},
		{"wrongLeft", "stdjava.EvaluationValue(d)", "stdjava.EvaluationValue(MON)", "generated expression"},
		{"duplicateLeft", "stdjava.EvaluationValue(d)", "func() *Eday { _ = stdjava.EvaluationValue(d); return stdjava.EvaluationValue(d) }()", "generated expression"},
		{"missingInitialization", "EdayJava2goEnsureInitialized(__java2goExecution); ", "", "generated expression"},
		{"duplicateInitialization", "EdayJava2goEnsureInitialized(__java2goExecution); ", "EdayJava2goEnsureInitialized(__java2goExecution); EdayJava2goEnsureInitialized(__java2goExecution); ", "generated expression"},
		{"wrongExecution", "EdayJava2goEnsureInitialized(__java2goExecution)", "EdayJava2goEnsureInitialized(stdjava.NewExecution())", "generated expression"},
		{"wrongOwner", "EdayJava2goEnsureInitialized(__java2goExecution)", "OtherJava2goEnsureInitialized(__java2goExecution)", "generated expression"},
		{"wrongSingleton", "return WED", "return MON", "generated expression"},
		{"erasedRHS", "func() *Eday", "func() any", "generated expression"},
		{"ordinalEquality", "stdjava.JavaReferenceEqual(stdjava.EvaluationValue(d), func() *Eday { EdayJava2goEnsureInitialized(__java2goExecution); return WED }())", "d.enumOrdinal == 2", "generated expression"},
		{"constantResult", "stdjava.JavaReferenceEqual(stdjava.EvaluationValue(d), func() *Eday { EdayJava2goEnsureInitialized(__java2goExecution); return WED }())", "true", "generated expression"},
		{"extraStatement", "return stdjava.JavaReferenceEqual", "_ = d; return stdjava.JavaReferenceEqual", "receiver guard and one return"},
		{"runtimeIdentity", "github.com/NickyBoy89/java2go/stdjava", "example.invalid/runtime", "runtime import"},
	} {
		t.Run(row.name, func(t *testing.T) {
			mutant := ci196MutateIdentityGuard(t, ci196QualifiedIdentityGuardSource, row.old, row.replacement)
			ciABI193Reject(t, row.message, func(report ciABI193Reporter) { ci196AssertQualifiedEnumIdentity(report, mutant) })
		})
	}
}

func TestCI196ClassLiteralASTGuards(t *testing.T) {
	ci196AssertDoubleClassIdentity(t, ci196DoubleClassIdentityGuardSource)
	alias := ci196MutateIdentityGuard(t, ci196DoubleClassIdentityGuardSource, `stdjava.TypeID("java.lang.Double")`, "stdjava.DoubleTypeID")
	ci196AssertDoubleClassIdentity(t, alias)
	for _, row := range []struct{ name, old, replacement, message string }{
		{"fieldType", "type0 *stdjava.Class", "type0 any", "generated expression"},
		{"fieldIdentity", "return gs.type0", "return gs.other", "generated expression"},
		{"operator", "gs.type0 ==", "gs.type0 !=", "class identity comparison"},
		{"primitive", `java.lang.Double`, `primitive:double`, "class literal descriptor"},
		{"integer", `java.lang.Double`, `java.lang.Integer`, "class literal descriptor"},
		{"wrongCallee", "stdjava.ClassLiteral", "stdjava.ObjectGetClass", "generated expression"},
		{"missingCall", `stdjava.ClassLiteral(stdjava.TypeID("java.lang.Double"))`, "gs.type0", "ClassLiteral call"},
		{"constantResult", `gs.type0 == stdjava.ClassLiteral(stdjava.TypeID("java.lang.Double"))`, "true", "class identity comparison"},
		{"receiverGuard", "if gs == nil { _ = *gs }", "if gs == nil {}", "receiver guard"},
		{"extraStatement", "return gs.type0", "_ = gs; return gs.type0", "receiver guard and one return"},
		{"runtimeIdentity", "github.com/NickyBoy89/java2go/stdjava", "example.invalid/runtime", "runtime import"},
	} {
		t.Run(row.name, func(t *testing.T) {
			mutant := ci196MutateIdentityGuard(t, ci196DoubleClassIdentityGuardSource, row.old, row.replacement)
			ciABI193Reject(t, row.message, func(report ciABI193Reporter) { ci196AssertDoubleClassIdentity(report, mutant) })
		})
	}
}
