package stdjava_test

import (
	j "github.com/NickyBoy89/java2go/stdjava"
	"testing"
)

type supplierDiagnosticMap = j.Map[*j.JavaString, *j.Integer]
type supplierDiagnosticExecution struct {
	value *supplierDiagnosticMap
	calls *int
	seen  **j.Execution
}

func (s supplierDiagnosticExecution) Get() *supplierDiagnosticMap {
	panic("public Get must lose to execution companion")
}
func (s supplierDiagnosticExecution) GetJava2goExecution(e *j.Execution) *supplierDiagnosticMap {
	*s.calls++
	*s.seen = e
	return s.value
}

type supplierDiagnosticPlain struct {
	value *supplierDiagnosticMap
	calls *int
}

func (s supplierDiagnosticPlain) Get() *supplierDiagnosticMap { *s.calls++; return s.value }

type supplierDiagnosticCounterfeit struct{ calls *int }

func (s supplierDiagnosticCounterfeit) Get() j.JavaMap {
	*s.calls++
	return j.NewTreeMap[*j.JavaString, *j.Integer]()
}
func (supplierDiagnosticCounterfeit) JavaDynamicTypeID() j.TypeID {
	return "java.util.function.Supplier"
}

func TestSupplierProjectedNativeMapFactoryAPIRED(t *testing.T) {
	e := j.NewExecution()
	original := j.NewTreeMap[*j.JavaString, *j.Integer]()
	calls := 0
	var seen *j.Execution
	factory := j.NewSupplierFuncAdapter[j.JavaMap](func(got *j.Execution) j.JavaMap { calls++; seen = got; return original })
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("existing native factory boundary panicked before success: get_calls=%d sameExecution=%v panic=%v", calls, seen == e, p)
		}
	}()
	out := j.StreamGroupingByDownstreamWith(j.NewStream(j.JavaStringFromHostUTF8("b"), j.JavaStringFromHostUTF8("a"), j.JavaStringFromHostUTF8("b")), func(v *j.JavaString) *j.JavaString { return v }, factory, func(s j.Stream[*j.JavaString]) *j.Integer { return j.BoxInteger(int32(s.Count())) }, e)
	if out != original || calls != 1 || seen != e {
		t.Fatalf("identity/calls/execution: same=%v calls=%d execution=%v", out == original, calls, seen == e)
	}
	keys := out.KeySet()
	if len(keys) != 2 || keys[0].CompareTo(j.JavaStringFromHostUTF8("a")) != 0 || keys[1].CompareTo(j.JavaStringFromHostUTF8("b")) != 0 {
		t.Fatalf("lost map-factory natural ordering: %v", keys)
	}
}

func TestSupplierProjectedLegacyFourPaths(t *testing.T) {
	for _, kind := range []string{"hidden", "public", "executionFunc", "plainFunc"} {
		t.Run(kind, func(t *testing.T) {
			e := j.NewExecution()
			original := j.NewTreeMap[*j.JavaString, *j.Integer]()
			calls := 0
			var seen *j.Execution
			var factory any
			switch kind {
			case "hidden":
				factory = supplierDiagnosticExecution{original, &calls, &seen}
			case "public":
				factory = supplierDiagnosticPlain{original, &calls}
			case "executionFunc":
				factory = func(got *j.Execution) *supplierDiagnosticMap { calls++; seen = got; return original }
			case "plainFunc":
				factory = func() *supplierDiagnosticMap { calls++; return original }
			}
			got := j.CallSupplierExecution[*supplierDiagnosticMap](e, factory)
			if got != original || calls != 1 {
				t.Fatalf("legacy identity/calls: %v %d", got == original, calls)
			}
			if (kind == "hidden" || kind == "executionFunc") && seen != e {
				t.Fatal("lost caller execution")
			}
		})
	}
}

func TestSupplierProjectedNativeRefusalControls(t *testing.T) {
	for _, kind := range []string{"noGet", "wrongResult", "counterfeit"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			var value any = struct{}{}
			if kind == "wrongResult" {
				value = j.NewSupplierFuncAdapter(func(*j.Execution) *j.JavaString { calls++; return j.JavaStringFromHostUTF8("wrong") })
			}
			if kind == "counterfeit" {
				value = supplierDiagnosticCounterfeit{&calls}
			}
			defer func() {
				p := recover()
				if p == nil || !j.CaughtAs(p, "UnsupportedOperationException") || calls != 0 {
					t.Fatalf("unproved supplier was called/admitted: calls=%d panic=%v", calls, p)
				}
			}()
			j.CallSupplierExecution[*supplierDiagnosticMap](j.NewExecution(), value)
		})
	}
}

func TestSupplierProjectedNullAndThrows(t *testing.T) {
	t.Run("nullSupplier", func(t *testing.T) {
		var s *j.SupplierFuncAdapter[*supplierDiagnosticMap]
		defer func() {
			if p := recover(); p == nil || !j.CaughtAs(p, "NullPointerException") {
				t.Fatalf("typed null supplier: %v", p)
			}
		}()
		j.CallSupplierExecution[*supplierDiagnosticMap](j.NewExecution(), s)
	})
	t.Run("nullProduct", func(t *testing.T) {
		calls := 0
		s := j.NewSupplierFuncAdapter(func(*j.Execution) *supplierDiagnosticMap { calls++; return nil })
		if j.CallSupplierExecution[*supplierDiagnosticMap](j.NewExecution(), s) != nil || calls != 1 {
			t.Fatal("null product changed or retried")
		}
	})
	t.Run("throwIdentity", func(t *testing.T) {
		calls := 0
		sentinel := &struct{ marker int }{17}
		s := j.NewSupplierFuncAdapter(func(*j.Execution) *supplierDiagnosticMap { calls++; panic(sentinel) })
		defer func() {
			if p := recover(); p != sentinel || calls != 1 {
				t.Fatalf("throw changed/retried: calls=%d panic=%v", calls, p)
			}
		}()
		j.CallSupplierExecution[*supplierDiagnosticMap](j.NewExecution(), s)
	})
}

func TestSupplierUnknownAdapterClassIdentityControl(t *testing.T) {
	s := j.NewSupplierFuncAdapter(func(*j.Execution) j.JavaMap { return j.NewTreeMap[*j.JavaString, *j.Integer]() })
	if id, known := j.ObjectDynamicType(s); known {
		t.Fatalf("unknown lambda implementation class fabricated: %q", id)
	}
	f := j.NewFunctionFuncAdapter(func(_ *j.Execution, x *j.JavaString) *j.JavaString { return x })
	id, known := j.ObjectDynamicType(f)
	t.Logf("existing Function convention, observed only: known=%v id=%q", known, id)
	defer func() {
		p := recover()
		if p == nil || !j.CaughtAs(p, "IllegalArgumentException") {
			t.Fatalf("unknown hidden class reflection was fabricated: %v", p)
		}
	}()
	j.ObjectGetClass(s)
}
