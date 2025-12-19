package main

import (
	"strings"
	"testing"
)

func TestAbstractIntegration_GeneratesStubAndTracksMetadata(t *testing.T) {
	src := `
    package abs.integration;
    public abstract class Shape {
        public abstract double area();
    }
    public class Square extends Shape {
        double side;
        public Square(double side) { this.side = side; }
        public double area() { return side * side; }
    }
    `

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
	if !strings.Contains(flat, "abstract method area not implemented") {
		t.Fatalf("expected stub to panic for abstract method, got:\n%s", out)
	}
	if !strings.Contains(flat, "*Square) Area() float64") {
		t.Fatalf("expected concrete override for Square.Area, got:\n%s", out)
	}
	if !strings.Contains(flat, "side * side") {
		t.Fatalf("expected concrete Area implementation to use side field, got:\n%s", out)
	}
}

func TestAbstractIntegration_ComplexHierarchyAndStubs(t *testing.T) {
	src := `
    package abs.integration;
    public abstract class BaseThing {
        protected int value;
        public BaseThing(int value) { this.value = value; }
        public int value() { return value; }
        public abstract String id();
        public abstract double compute(double a, double b);
    }

    public abstract class MidThing extends BaseThing {
        protected String name;
        public MidThing(int value, String name) { super(value); this.name = name; }
        public String label() { return name + "-" + value(); }
        public abstract String id();
    }

    public class ConcreteThing extends MidThing {
        public ConcreteThing(int value, String name) { super(value, name); }
        public String id() { return name; }
        public double compute(double a, double b) { return a * b + value(); }
        public String label() { return "override-" + super.label(); }
    }
    `

	helper := setupParseHelper(t, src)

	baseScope := helper.File.Symbols.FindClassScope("BaseThing")
	if baseScope == nil || !baseScope.IsAbstract {
		t.Fatalf("expected BaseThing to exist and be abstract in symbols")
	}

	midScope := helper.File.Symbols.FindClassScope("MidThing")
	if midScope == nil || !midScope.IsAbstract {
		t.Fatalf("expected MidThing to exist and be abstract in symbols")
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

	if !strings.Contains(flat, "*MidThing) Id() string") {
		t.Fatalf("expected MidThing.Id abstract stub in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "func (mg *MidThing) Label() string") {
		t.Fatalf("expected MidThing.label concrete method to be emitted, got:\n%s", out)
	}

	if !strings.Contains(flat, "*ConcreteThing) Id() string") {
		t.Fatalf("expected ConcreteThing.Id concrete override in output, got:\n%s", out)
	}
	if !strings.Contains(flat, "return name") {
		t.Fatalf("expected ConcreteThing.Id to return the name field, got:\n%s", out)
	}
	if !strings.Contains(flat, "*ConcreteThing) Compute(a float64, b float64) float64") {
		t.Fatalf("expected ConcreteThing.Compute to be concrete implementation, got:\n%s", out)
	}
	if !strings.Contains(flat, "\"override-\"") {
		t.Fatalf("expected ConcreteThing.Label to include override marker, got:\n%s", out)
	}
}
