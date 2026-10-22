package stdjava_test

import (
	j "github.com/NickyBoy89/java2go/stdjava"
	"testing"
)

func TestSupplierReferenceCanonicalMembershipAndProjection(t *testing.T) {
	e := j.NewExecution()
	original := j.NewTreeMap[*j.JavaString, *j.Integer]()
	calls := 0
	s := j.NewSupplierFuncAdapter[j.JavaMap](func(got *j.Execution) j.JavaMap {
		calls++
		if got != e {
			t.Fatal("erased callback lost consumer Execution")
		}
		return original
	})
	if !j.ObjectInstanceOf(s, j.SupplierTypeID) || !j.ClassLiteral(j.SupplierTypeID).IsInstance(s) || j.ObjectInstanceOf(s, "Supplier") {
		t.Fatal("canonical membership confused implementation class or source homonym")
	}
	if _, known := j.ObjectDynamicType(s); known {
		t.Fatal("unknown hidden implementation class fabricated")
	}
	if got := j.CallSupplierProjectedExecution[*supplierDiagnosticMap](e, s, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer]); got != original || calls != 1 {
		t.Fatal("erased result copied or invoked twice")
	}
}

func TestSupplierReferenceNullThrowsAndUnsupportedProducts(t *testing.T) {
	t.Run("typedNullSupplier", func(t *testing.T) {
		var s *j.SupplierFuncAdapter[j.JavaMap]
		defer func() {
			if p := recover(); p == nil || !j.CaughtAs(p, "NullPointerException") {
				t.Fatalf("typed null: %v", p)
			}
		}()
		j.CallSupplierProjectedExecution[*supplierDiagnosticMap](j.NewExecution(), s, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	})
	t.Run("typedNullProduct", func(t *testing.T) {
		calls := 0
		var product *supplierDiagnosticMap
		s := j.NewSupplierFuncAdapter[j.JavaMap](func(*j.Execution) j.JavaMap { calls++; return product })
		if got := j.CallSupplierProjectedExecution[*supplierDiagnosticMap](j.NewExecution(), s, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer]); got != nil || calls != 1 {
			t.Fatal("typed null product changed or retried")
		}
	})
	t.Run("throwIdentity", func(t *testing.T) {
		calls := 0
		sentinel := &struct{ marker int }{19}
		s := j.NewSupplierFuncAdapter[j.JavaMap](func(*j.Execution) j.JavaMap { calls++; panic(sentinel) })
		defer func() {
			if p := recover(); p != sentinel || calls != 1 {
				t.Fatalf("throw identity/retry: %v %d", p, calls)
			}
		}()
		j.CallSupplierProjectedExecution[*supplierDiagnosticMap](j.NewExecution(), s, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	})
	t.Run("nativeCarrierMismatch", func(t *testing.T) {
		original := j.NewMap[any, any]()
		calls := 0
		s := j.NewSupplierFuncAdapter[j.JavaMap](func(*j.Execution) j.JavaMap { calls++; return original })
		defer func() {
			p := recover()
			if p == nil || !j.CaughtAs(p, "UnsupportedOperationException") || calls != 1 || original.Size() != 0 {
				t.Fatalf("carrier was guessed/copied/retried: %v %d", p, calls)
			}
		}()
		j.CallSupplierProjectedExecution[*supplierDiagnosticMap](j.NewExecution(), s, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	})
	t.Run("wrongNominalProduct", func(t *testing.T) {
		calls := 0
		s := j.NewSupplierFuncAdapter(func(*j.Execution) any { calls++; return j.BoxInteger(7) })
		defer func() {
			p := recover()
			if p == nil || !j.CaughtAs(p, "ClassCastException") || calls != 1 {
				t.Fatalf("wrong nominal product: %v %d", p, calls)
			}
		}()
		j.CallSupplierProjectedExecution[*supplierDiagnosticMap](j.NewExecution(), s, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	})
	t.Run("counterfeitPublicShape", func(t *testing.T) {
		calls := 0
		value := supplierDiagnosticCounterfeit{&calls}
		defer func() {
			if p := recover(); p == nil || calls != 0 {
				t.Fatalf("public shape acquired sealed role: %v %d", p, calls)
			}
		}()
		j.CallSupplierProjectedExecution[*supplierDiagnosticMap](j.NewExecution(), value, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	})
}

type supplierSourceDiagnosticRoot struct{ *j.ObjectInfo }

func TestSupplierReferenceSourceOwnershipAndIdentity(t *testing.T) {
	const id j.TypeID = "supplier.reference.control.Source"
	j.RegisterJavaType(id, j.ObjectTypeID, j.SupplierTypeID)
	j.RegisterJavaSourceType(id)
	e := j.NewExecution()
	original := j.NewTreeMap[*j.JavaString, *j.Integer]()
	calls := 0
	var info *j.ObjectInfo
	var view *j.SupplierSourceView[j.JavaMap]
	info = j.NewObjectInfo(id, func(requested j.TypeID) any {
		if requested == j.SupplierTypeID {
			return view
		}
		return nil
	})
	view = j.NewSupplierSourceView(info, func(got *j.Execution) j.JavaMap {
		calls++
		if got != e {
			t.Fatal("source callback lost consuming Execution")
		}
		return original
	})
	root := &supplierSourceDiagnosticRoot{info}
	got := j.CallSupplierProjectedExecution[*supplierDiagnosticMap](e, root, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	if got != original || calls != 1 || view.JavaObjectInfo() != info || j.ObjectGetClass(view).TypeID() != id || !j.JavaReferenceEqual(view, root) {
		t.Fatal("source allocation identity changed")
	}
	t.Run("detachedView", func(t *testing.T) {
		detachedInfo := j.NewObjectInfo(id, nil)
		detached := j.NewSupplierSourceView(detachedInfo, func(*j.Execution) j.JavaMap { t.Fatal("detached get invoked"); return original })
		defer func() {
			if p := recover(); p == nil || !j.CaughtAs(p, "UnsupportedOperationException") {
				t.Fatalf("detached source view admitted: %v", p)
			}
		}()
		j.CallSupplierProjectedExecution[*supplierDiagnosticMap](e, detached, j.SupplierNativeMapProduct[*j.JavaString, *j.Integer])
	})
	t.Run("wrongProviderIdentity", func(t *testing.T) {
		other := j.NewObjectInfo(id, func(j.TypeID) any { return view })
		defer func() {
			if p := recover(); p == nil || !j.CaughtAs(p, "UnsupportedOperationException") {
				t.Fatalf("provider identity donation: %v", p)
			}
		}()
		j.SupplierGetReferenceExecution(e, &supplierSourceDiagnosticRoot{other})
	})
	t.Run("wrongSourceDescriptor", func(t *testing.T) {
		const wrong j.TypeID = "supplier.reference.control.Supplier"
		j.RegisterJavaType(wrong, j.ObjectTypeID)
		j.RegisterJavaSourceType(wrong)
		defer func() {
			if p := recover(); p == nil || !j.CaughtAs(p, "UnsupportedOperationException") {
				t.Fatalf("source homonym admitted: %v", p)
			}
		}()
		j.NewSupplierSourceView(j.NewObjectInfo(wrong, nil), func(*j.Execution) j.JavaMap { t.Fatal("constructor invoked get"); return original })
	})
}

func TestSupplierReferenceProjectedLegacyFourPaths(t *testing.T) {
	for _, kind := range []string{"hidden", "public", "executionFunc", "plainFunc"} {
		t.Run(kind, func(t *testing.T) {
			e := j.NewExecution()
			original := j.NewMap[*j.JavaString, *j.Integer]()
			calls := 0
			var seen *j.Execution
			var value any
			switch kind {
			case "hidden":
				value = supplierDiagnosticExecution{original, &calls, &seen}
			case "public":
				value = supplierDiagnosticPlain{original, &calls}
			case "executionFunc":
				value = func(got *j.Execution) *supplierDiagnosticMap { calls++; seen = got; return original }
			case "plainFunc":
				value = func() *supplierDiagnosticMap { calls++; return original }
			}
			got := j.CallSupplierProjectedExecution(e, value, func(any) *supplierDiagnosticMap { t.Fatal("legacy exact path projected"); return nil })
			if got != original || calls != 1 || ((kind == "hidden" || kind == "executionFunc") && seen != e) {
				t.Fatal("legacy exact path changed")
			}
		})
	}
}

func TestSupplierReferenceFactoryComparatorEquivalence(t *testing.T) {
	e := j.NewExecution()
	comparisons, calls := 0, 0
	original := j.NewTreeMapWith[*j.JavaString, *j.Integer](func(a, b *j.JavaString) int32 {
		comparisons++
		return a.Length() - b.Length()
	})
	factory := j.NewSupplierFuncAdapter[j.JavaMap](func(got *j.Execution) j.JavaMap {
		calls++
		if got != e {
			t.Fatal("factory execution changed")
		}
		return original
	})
	x, y := j.JavaStringFromHostUTF8("x"), j.JavaStringFromHostUTF8("y")
	if j.ObjectsEqual(x, y, e) {
		t.Fatal("control keys must differ under equals")
	}
	out := j.StreamGroupingByDownstreamWith(j.NewStream(x, y), func(v *j.JavaString) *j.JavaString { return v }, factory, func(s j.Stream[*j.JavaString]) *j.Integer { return j.BoxInteger(int32(s.Count())) }, e)
	if out != original || calls != 1 || comparisons == 0 || out.Size() != 1 || !j.ObjectsEqual(out.Get(x, e), j.BoxInteger(2), e) {
		t.Fatal("factory identity/comparator equivalence was replaced by equals grouping")
	}
}
