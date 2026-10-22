package transpiler

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestNamePath168FrozenWorkBudgets(t *testing.T) {
	t.Run("derived", func(t *testing.T) {
		var source strings.Builder
		for i := 0; i < 128; i++ {
			fmt.Fprintf(&source, "class Scope%03d {}\n", i)
		}
		helper := setupParseHelper(t, source.String())
		work := func() {
			for i := 0; i < 16; i++ {
				if generatedIdentifierExists(fmt.Sprintf("Absent%03dJava2goExecution", i), helper.Ctx.currentClass) {
					t.Fatal("negative collision")
				}
			}
		}
		r := namePath168Measure(t, "derived-negative-128x16", work)
		if r.Cgo != 0 {
			t.Errorf("derived Cgo changed: %d", r.Cgo)
		}
		if r.Allocs > 1446 || r.Bytes > 21522 {
			t.Errorf("FROZEN_WORK_RED derived: allocs %.0f >1446 or bytes %d >21522", r.Allocs, r.Bytes)
		}
		runtime.KeepAlive(helper)
	})
	t.Run("hierarchy", func(t *testing.T) {
		var source strings.Builder
		source.WriteString("class H0 {}\n")
		for i := 1; i <= 8; i++ {
			fmt.Fprintf(&source, "class H%d extends H%d {}\n", i, i-1)
		}
		helper := setupParseHelper(t, source.String())
		owner := helper.File.Symbols.FindClassScope("H8")
		ctx := helper.Ctx.Clone()
		ctx.currentClass = owner
		work := func() {
			for i := 0; i < 8; i++ {
				if memberTypeInHierarchy(owner, "Absent", ctx) != nil {
					t.Fatal("negative hierarchy result")
				}
			}
		}
		r := namePath168Measure(t, "hierarchy-depth8x8", work)
		if r.Cgo != 64 {
			t.Errorf("parent-resolution Cgo changed: %d instead of64", r.Cgo)
		}
		if r.Allocs > 256 || r.Bytes > 8192 {
			t.Errorf("FROZEN_WORK_RED hierarchy: allocs %.0f >256 or bytes %d >8192", r.Allocs, r.Bytes)
		}
		if len(ctx.memberTypeLookupPath) != 0 {
			t.Fatal("work escaped caller path")
		}
		runtime.KeepAlive(helper)
	})
}

func TestNamePath168GlobalEnumAndRetainedContext(t *testing.T) {
	helper := setupParseHelper(t, "class Root { int field; void method(){} } enum E { ITEM }")
	ctx := helper.Ctx.Clone()
	seed := map[*symbol.ClassScope]bool{ctx.currentClass: false}
	ctx.memberTypeLookupPath = seed
	retained := classScopeCtx(ctx.currentClass, ctx)
	memberTypeInHierarchy(ctx.currentClass, "Absent", ctx)
	if len(seed) != 1 || seed[ctx.currentClass] || len(retained.memberTypeLookupPath) != 1 {
		t.Fatal("retained context state changed")
	}
	query := "LateGlobalJava2goExecution"
	check := func() {
		t.Helper()
		if generatedIdentifierExists(query, nil) != namePath168OldGeneratedIdentifierExists(query, nil) {
			t.Fatal("global old-oracle mismatch")
		}
	}
	check()
	late := namePath168Parse(t, "Late.java", "package late; class Late {}")
	late.Symbols.BaseClass.Class.Name = query
	check()
	if !generatedIdentifierExists(query, nil) {
		t.Fatal("late global invisible")
	}
	late.Symbols.BaseClass.Class.Name = "Renamed"
	check()
	if generatedIdentifierExists(query, nil) {
		t.Fatal("renamed global retained")
	}
	enum := helper.File.Symbols.FindClassScope("E")
	if enum == nil || len(enum.EnumConstants) != 1 {
		t.Fatal("missing real enum")
	}
	original := enum.EnumConstants[0].Field.Name
	enum.EnumConstants[0].Field.Name = query
	check()
	if !generatedIdentifierExists(query, nil) {
		t.Fatal("late enum invisible")
	}
	enum.EnumConstants[0].Field.Name = original
	check()
	if generatedIdentifierExists(query, nil) {
		t.Fatal("renamed enum retained")
	}
	delete(symbol.GlobalScope.Packages, "late")
	check()
	runtime.KeepAlive(helper)
	runtime.KeepAlive(late)
	runtime.KeepAlive(retained)
}
