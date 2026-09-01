package transpiler

import (
	"strings"
	"testing"
)

func TestGapFeaturesStreamsClassLiteralsAndMultiFields(t *testing.T) {
	src := `
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

	out := renderGoFileFromJava(t, src)
	for _, fragment := range []string{
		"type GapFeatures[T stdjava.JavaNumber] struct",
		"x\tfloat64",
		"y\tfloat64",
		"stdjava.NumberDoubleValue(n)",
		"stdjava.ClassLiteral(stdjava.DoubleTypeID)",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("generated source is missing %q:\n%s", fragment, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "math"
    "testing"

    "github.com/NickyBoy89/java2go/stdjava"
)

func TestGapFeaturesRuntime(t *testing.T) {
    program := NewGapFeatures[float64](
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
    got := program.Process(stdjava.NewListFrom[float64](-1, 4))
    values := got.Slice()
    if len(values) != 1 || values[0] != 2*math.Pi {
        t.Fatalf("Process() = %v, want [%v]", values, 2*math.Pi)
    }
}
`)
}

func TestGapFeaturesGenericEnumMethodHelper(t *testing.T) {
	src := `
public enum GenericOperation {
    DOUBLE {
        @Override
        public <T extends Number> double apply(T value) {
            return value.doubleValue() * 2;
        }
    };

    public abstract <T extends Number> double apply(T value);
}
`

	out := renderGoFileFromJava(t, src)
	for _, fragment := range []string{
		"type GenericOperationApplyHelper[T stdjava.JavaNumber] struct",
		"func NewGenericOperationApplyHelper[T stdjava.JavaNumber]",
		"_GenericOperation_DOUBLE_Apply[T]",
		"stdjava.NumberDoubleValue(value)",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("generated enum source is missing %q:\n%s", fragment, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestGenericEnumRuntime(t *testing.T) {
    helper := NewGenericOperationApplyHelper[float64](DOUBLE)
    if got := helper.Apply(2.5); got != 5 {
        t.Fatalf("DOUBLE.apply(2.5) = %v, want 5", got)
    }
}
`)
}
