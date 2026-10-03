package transpiler

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func loadJavaTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test java file %s: %v", path, err)
	}
	return string(data)
}

func TestAbstractIntegration_GeneratesStubAndTracksMetadata(t *testing.T) {
	src := loadJavaTestFile(t, "../testfiles/abstract/ShapeHierarchy.java")

	helper := setupParseHelper(t, src)
	shapeScope := helper.File.Symbols.FindClassScope("Shape")
	if shapeScope == nil {
		t.Fatalf("expected Shape scope to be present")
	}
	if !shapeScope.IsAbstract {
		t.Fatalf("expected Shape to be marked abstract in symbols")
	}

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "*Shape) Area() float64") {
		t.Fatalf("expected abstract method stub on Shape, got:\n%s", out)
	}
	if !strings.Contains(flat, "*Shape) Perimeter() float64") {
		t.Fatalf("expected perimeter abstract stub on Shape, got:\n%s", out)
	}
	if !strings.Contains(flat, "abstract method area not implemented") {
		t.Fatalf("expected stub to panic for abstract method, got:\n%s", out)
	}
	if !strings.Contains(flat, "abstract method perimeter not implemented") {
		t.Fatalf("expected stub to panic for abstract method, got:\n%s", out)
	}
	if !strings.Contains(flat, "*Square) Area() float64") {
		t.Fatalf("expected concrete override for Square.Area, got:\n%s", out)
	}
	if !strings.Contains(flat, "se.side * se.side") {
		t.Fatalf("expected concrete Area implementation to use side field, got:\n%s", out)
	}
	if !strings.Contains(flat, "*Circle) Perimeter() float64") {
		t.Fatalf("expected concrete Circle.Perimeter implementation, got:\n%s", out)
	}
}

func TestAbstractIntegration_ComplexHierarchyAndStubs(t *testing.T) {
	src := loadJavaTestFile(t, "../testfiles/abstract/ComplexAbstractHierarchy.java")

	helper := setupParseHelper(t, src)

	baseScope := helper.File.Symbols.FindClassScope("BaseThing")
	if baseScope == nil || !baseScope.IsAbstract {
		t.Fatalf("expected BaseThing to exist and be abstract in symbols")
	}

	midScope := helper.File.Symbols.FindClassScope("MidThing")
	if midScope == nil || !midScope.IsAbstract {
		t.Fatalf("expected MidThing to exist and be abstract in symbols")
	}

	leafScope := helper.File.Symbols.FindClassScope("LeafThing")
	if leafScope == nil || !leafScope.IsAbstract {
		t.Fatalf("expected LeafThing to exist and be abstract in symbols")
	}

	concreteScope := helper.File.Symbols.FindClassScope("ConcreteThing")
	if concreteScope == nil {
		t.Fatalf("expected ConcreteThing to exist in symbols")
	}
	if concreteScope.IsAbstract {
		t.Fatalf("expected ConcreteThing to be concrete, but it was marked abstract")
	}

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "*BaseThing) Id() *stdjava.JavaString") {
		t.Fatalf("expected BaseThing.Id abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "*BaseThing) Compute(a float64, b float64) float64") {
		t.Fatalf("expected BaseThing.Compute abstract stub in output, got:\n%s", out)
	}
	if strings.Count(flat, "abstract method") < 2 {
		t.Fatalf("expected abstract stubs to include panic messages for BaseThing methods, got:\n%s", out)
	}
	if !strings.Contains(flat, "*BaseThing) Describe() *stdjava.JavaString") {
		t.Fatalf("expected BaseThing.Describe concrete method in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "return stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(__java2goExecution, bg.Java2goBaseThingSelf.IdJava2goExecution(__java2goExecution)), stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{58})))), stdjava.JavaStringValueOfInt(int32(bg.Value0)))") {
		t.Fatalf("expected BaseThing.Describe to use Id() and value field, got:\n%s", out)
	}

	if !strings.Contains(flat, "*MidThing) Id() *stdjava.JavaString") {
		t.Fatalf("expected MidThing.Id abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "*MidThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected MidThing.Combine abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "func (mg *MidThing) Label() *stdjava.JavaString") {
		t.Fatalf("expected MidThing.label concrete method to be emitted, got:\n%s", out)
	}

	if !strings.Contains(flat, "*LeafThing) Describe() *stdjava.JavaString") {
		t.Fatalf("expected LeafThing.Describe override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "__java2goInvocationReceiver := lg.MidThing") ||
		!strings.Contains(flat, "return __java2goExecutionReceiver.DescribeJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected LeafThing.Describe to call super.describe(), got:\n%s", out)
	}
	if strings.Contains(flat, "*LeafThing) Id() *stdjava.JavaString") {
		t.Fatalf("expected LeafThing to rely on inherited stubs for Id, got:\n%s", out)
	}
	if strings.Contains(flat, "*LeafThing) Compute(a float64, b float64) float64") {
		t.Fatalf("expected LeafThing to rely on inherited stubs for Compute, got:\n%s", out)
	}
	if strings.Contains(flat, "*LeafThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected LeafThing to rely on inherited stubs for Combine, got:\n%s", out)
	}

	if !strings.Contains(flat, "*ConcreteThing) Id() *stdjava.JavaString") {
		t.Fatalf("expected ConcreteThing.Id concrete override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "return stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{99, 111, 110, 99, 114, 101, 116, 101, 45})), stdjava.JavaStringTextOperandExecution(__java2goExecution, cg.Name))") {
		t.Fatalf("expected ConcreteThing.Id to return the name field, got:\n%s", out)
	}
	if !strings.Contains(flat, "*ConcreteThing) Compute(a float64, b float64) float64") {
		t.Fatalf("expected ConcreteThing.Compute to be concrete implementation, got:\n%s", out)
	}
	if !strings.Contains(flat, "*ConcreteThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected ConcreteThing.Combine to be concrete implementation, got:\n%s", out)
	}
	if !strings.Contains(flat, "total := first + second + third") {
		t.Fatalf("expected ConcreteThing.Combine to declare total, got:\n%s", out)
	}
	if !strings.Contains(flat, "return stdjava.EvaluationValue[float64](total) + cg.ComputeJava2goExecution(__java2goExecution, stdjava.EvaluationValue[float64](total), float64(cg.Java2goBaseThingSelf.ValueJava2goExecution(__java2goExecution)))") {
		t.Fatalf("expected ConcreteThing.Combine to call compute/value, got:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.JavaStringLiteralUTF16([]uint16{111, 118, 101, 114, 114, 105, 100, 101, 45})") {
		t.Fatalf("expected ConcreteThing.Label to include override marker, got:\n%s", out)
	}
	if !strings.Contains(flat, "__java2goInvocationReceiver := cg.LeafThing") ||
		!strings.Contains(flat, "return __java2goExecutionReceiver.LabelJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected ConcreteThing.Label to call super.label(), got:\n%s", out)
	}
	if !strings.Contains(flat, "*AltConcreteThing) Id() *stdjava.JavaString") {
		t.Fatalf("expected AltConcreteThing.Id concrete override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "*AltConcreteThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected AltConcreteThing.Combine concrete implementation, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewBaseThing(value int32) *BaseThing") {
		t.Fatalf("expected BaseThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "bg.Value0 = value") {
		t.Fatalf("expected BaseThing constructor to initialize value field, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewMidThing(value int32, name *stdjava.JavaString) *MidThing") {
		t.Fatalf("expected MidThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "mg.BaseThing = NewBaseThingJava2goWithSelfJava2goExecution(__java2goExecution, __java2goMostDerived, value)") {
		t.Fatalf("expected MidThing constructor to call BaseThing constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "mg.Name = name") {
		t.Fatalf("expected MidThing constructor to initialize name field, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewLeafThing(value int32, name *stdjava.JavaString) *LeafThing") {
		t.Fatalf("expected LeafThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "lg.MidThing = NewMidThingJava2goWithSelfJava2goExecution(__java2goExecution, __java2goMostDerived, value, name)") {
		t.Fatalf("expected LeafThing constructor to call MidThing constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewConcreteThing(value int32, name *stdjava.JavaString) *ConcreteThing") {
		t.Fatalf("expected ConcreteThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "cg.LeafThing = NewLeafThingJava2goWithSelfJava2goExecution(__java2goExecution, __java2goMostDerived, value, name)") {
		t.Fatalf("expected ConcreteThing constructor to call LeafThing constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewAltConcreteThing(value int32, name *stdjava.JavaString) *AltConcreteThing") {
		t.Fatalf("expected AltConcreteThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "ag.MidThing = NewMidThingJava2goWithSelfJava2goExecution(__java2goExecution, __java2goMostDerived, value, name)") {
		t.Fatalf("expected AltConcreteThing constructor to call MidThing constructor, got:\n%s", out)
	}
}

// The historical parser fixture groups several public top-level declarations.
// Make only their file layout legal for javac; keep the hierarchy and behavior
// intact while validating the exported protected-member Go ABI.
func TestAbstractIntegrationProtectedHierarchyJVMParity(t *testing.T) {
	source := loadJavaTestFile(t, "../testfiles/abstract/ComplexAbstractHierarchy.java")
	source = strings.Replace(source, "package abs.integration.complex;", "", 1)
	source = strings.ReplaceAll(source, "public abstract class ", "abstract class ")
	source = strings.ReplaceAll(source, "public class ", "class ")
	source += `public class AbstractHierarchyOracle {
  public static String run() {
   ConcreteThing value=new ConcreteThing(7,"ink");
   MidThing other=new AltConcreteThing(3,"paper");
   return value.describe()+":"+value.label()+":"+value.combine(1,2,3)
    +":"+other.describe()+":"+other.label()+":"+other.combine(6,2,1);
  }
 }`
	want := campaignRuntimeJavaOracle(t, "AbstractHierarchyOracle", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
)
func TestHierarchy(t *testing.T) {
 const want = %q
 got := Run()
 if got == nil { t.Fatal("Run() returned null") }
 units := got.UTF16Copy()
 if !slices.Equal(units, utf16.Encode([]rune(want))) {
  t.Fatalf("got %%q want %%q", string(utf16.Decode(units)), want)
 }
}`, want))
}
