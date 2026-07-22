package transpiler

import (
	"strings"
	"testing"
)

func TestStaticWildcardCaptureSupportsInvariantGenericArguments(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class StaticWildcardCaptureProgram {
    interface Root {
        int code();
    }

    static class Impl implements Root {
        int value;

        Impl(int value) {
            this.value = value;
        }

        public int code() {
            return value;
        }
    }

    static class Box<T extends Root> {
        T value;

        Box(T value) {
            this.value = value;
        }

        T get() {
            return value;
        }
    }

    static <FirstT extends Root> int read(
            FirstT marker,
            Root SecondT,
            Box<? extends Root> first,
            Box<?> second,
            Box<? super Impl> third) {
        return marker.code() * 1000
                + first.get().code() * 100
                + second.get().code() * 10
                + third.get().code();
    }

    public static String run() {
        Impl marker = new Impl(1);
        Box<Impl> first = new Box<Impl>(new Impl(2));
        Box<Impl> second = new Box<Impl>(new Impl(3));
        Box<Root> third = new Box<Root>(new Impl(4));
        return "" + read(marker, marker, first, second, third);
    }
}
`)

	flat := normalizeSpaces(out)
	for _, expected := range []string{
		"FirstT2 StaticWildcardCaptureProgramroot",
		"SecondT2 StaticWildcardCaptureProgramroot",
		"ThirdT StaticWildcardCaptureProgramroot",
		"first *StaticWildcardCaptureProgrambox[FirstT2]",
		"second *StaticWildcardCaptureProgrambox[SecondT2]",
		"third *StaticWildcardCaptureProgrambox[ThirdT]",
	} {
		if !strings.Contains(flat, expected) {
			t.Fatalf("generated wildcard capture missing %q:\n%s", expected, out)
		}
	}
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestStaticWildcardCaptureRuntime(t *testing.T) {
    if got := Run(); got != "1234" {
        t.Fatalf("Run() = %q, want 1234", got)
    }
}
`)
}

func TestStaticWildcardCaptureReferenceArrayUsesReadableUpperProjection(t *testing.T) {
	helper := setupParseHelper(t, `
public class StaticWildcardArrayProgram {
    interface Root {}
    static class Box<T extends Root> {}
    static void accept(Box<?>[][] values) {}
}
`)

	scope := helper.File.Symbols.FindClassScope("StaticWildcardArrayProgram")
	if scope == nil {
		t.Fatal("StaticWildcardArrayProgram scope was not parsed")
	}
	methods := scope.FindMethod().ByOriginalName("accept")
	if len(methods) != 1 {
		t.Fatalf("accept definitions = %d, want 1", len(methods))
	}
	ctx := helper.Ctx
	ctx.currentClass = scope
	synthetic, rewritten := synthesizeRawGenericFunctionParameters(methods[0], ctx)
	if len(synthetic) != 0 {
		t.Fatalf("reference-array wildcard synthesized uninferable parameters: %#v", synthetic)
	}
	if got, want := rewritten["values"], "Box<Root>[][]"; got != want {
		t.Fatalf("rewritten array type = %q, want %q", got, want)
	}
}
