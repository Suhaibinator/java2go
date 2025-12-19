package main

import (
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
	src := loadJavaTestFile(t, "testfiles/abstract/ShapeHierarchy.java")

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
	src := loadJavaTestFile(t, "testfiles/abstract/ComplexAbstractHierarchy.java")

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

	if !strings.Contains(flat, "*BaseThing) Id() string") {
		t.Fatalf("expected BaseThing.Id abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "*BaseThing) Compute(a float64, b float64) float64") {
		t.Fatalf("expected BaseThing.Compute abstract stub in output, got:\n%s", out)
	}
	if strings.Count(flat, "abstract method") < 2 {
		t.Fatalf("expected abstract stubs to include panic messages for BaseThing methods, got:\n%s", out)
	}
	if !strings.Contains(flat, "*BaseThing) Describe() string") {
		t.Fatalf("expected BaseThing.Describe concrete method in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "return bg.Id() + \":\" + bg.value") {
		t.Fatalf("expected BaseThing.Describe to use Id() and value field, got:\n%s", out)
	}

	if !strings.Contains(flat, "*MidThing) Id() string") {
		t.Fatalf("expected MidThing.Id abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "*MidThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected MidThing.Combine abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "func (mg *MidThing) Label() string") {
		t.Fatalf("expected MidThing.label concrete method to be emitted, got:\n%s", out)
	}

	if !strings.Contains(flat, "*LeafThing) Describe() string") {
		t.Fatalf("expected LeafThing.Describe override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "return \"leaf-\" + lg.MidThing.Describe()") {
		t.Fatalf("expected LeafThing.Describe to call super.describe(), got:\n%s", out)
	}
	if strings.Contains(flat, "*LeafThing) Id() string") {
		t.Fatalf("expected LeafThing to rely on inherited stubs for Id, got:\n%s", out)
	}
	if strings.Contains(flat, "*LeafThing) Compute(a float64, b float64) float64") {
		t.Fatalf("expected LeafThing to rely on inherited stubs for Compute, got:\n%s", out)
	}
	if strings.Contains(flat, "*LeafThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected LeafThing to rely on inherited stubs for Combine, got:\n%s", out)
	}

	if !strings.Contains(flat, "*ConcreteThing) Id() string") {
		t.Fatalf("expected ConcreteThing.Id concrete override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "return \"concrete-\" + cg.name") {
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
	if !strings.Contains(flat, "return total + cg.Compute(total, cg.Value())") {
		t.Fatalf("expected ConcreteThing.Combine to call compute/value, got:\n%s", out)
	}
	if !strings.Contains(flat, "\"override-\"") {
		t.Fatalf("expected ConcreteThing.Label to include override marker, got:\n%s", out)
	}
	if !strings.Contains(flat, "return \"override-\" + cg.LeafThing.Label()") {
		t.Fatalf("expected ConcreteThing.Label to call super.label(), got:\n%s", out)
	}
	if !strings.Contains(flat, "*AltConcreteThing) Id() string") {
		t.Fatalf("expected AltConcreteThing.Id concrete override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "*AltConcreteThing) Combine(first float64, second float64, third float64) float64") {
		t.Fatalf("expected AltConcreteThing.Combine concrete implementation, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewBaseThing(value int32) *BaseThing") {
		t.Fatalf("expected BaseThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "bg.value = value") {
		t.Fatalf("expected BaseThing constructor to initialize value field, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewMidThing(value int32, name string) *MidThing") {
		t.Fatalf("expected MidThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "mg.BaseThing = NewBaseThing(value)") {
		t.Fatalf("expected MidThing constructor to call BaseThing constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "mg.name = name") {
		t.Fatalf("expected MidThing constructor to initialize name field, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewLeafThing(value int32, name string) *LeafThing") {
		t.Fatalf("expected LeafThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "lg.MidThing = NewMidThing(value, name)") {
		t.Fatalf("expected LeafThing constructor to call MidThing constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewConcreteThing(value int32, name string) *ConcreteThing") {
		t.Fatalf("expected ConcreteThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "cg.LeafThing = NewLeafThing(value, name)") {
		t.Fatalf("expected ConcreteThing constructor to call LeafThing constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewAltConcreteThing(value int32, name string) *AltConcreteThing") {
		t.Fatalf("expected AltConcreteThing constructor to be emitted, got:\n%s", out)
	}
	if !strings.Contains(flat, "ag.MidThing = NewMidThing(value, name)") {
		t.Fatalf("expected AltConcreteThing constructor to call MidThing constructor, got:\n%s", out)
	}
}
