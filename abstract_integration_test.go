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
