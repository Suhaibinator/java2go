package stdjava

// IntPredicate retains Java reference identity while carrying its primitive SAM.
type IntPredicate interface{ Test(int32) bool }

type IntPredicateFuncAdapter struct {
	*ObjectInfo
	call     func(*Execution, int32) bool
	defaults IntPredicateSourceDefaults
}

// Source defaults are retained by exact declaration-selected callbacks. An
// absent callback uses the JDK default; a source override may accept null.
type IntPredicateSourceDefaults struct {
	And    func(*Execution, IntPredicate) IntPredicate
	Or     func(*Execution, IntPredicate) IntPredicate
	Negate func(*Execution) IntPredicate
}

func NewIntPredicateSourceViewWithDefaults(info *ObjectInfo, call func(*Execution, int32) bool, defaults IntPredicateSourceDefaults) *IntPredicateFuncAdapter {
	return &IntPredicateFuncAdapter{ObjectInfo: info, call: call, defaults: defaults}
}
func (f *IntPredicateFuncAdapter) intPredicateAndOverride(e *Execution, other IntPredicate) (IntPredicate, bool) {
	if f.defaults.And != nil {
		return f.defaults.And(e, other), true
	}
	return nil, false
}
func (f *IntPredicateFuncAdapter) intPredicateOrOverride(e *Execution, other IntPredicate) (IntPredicate, bool) {
	if f.defaults.Or != nil {
		return f.defaults.Or(e, other), true
	}
	return nil, false
}
func (f *IntPredicateFuncAdapter) intPredicateNegateOverride(e *Execution) (IntPredicate, bool) {
	if f.defaults.Negate != nil {
		return f.defaults.Negate(e), true
	}
	return nil, false
}

func NewIntPredicateFuncAdapter(call func(*Execution, int32) bool) *IntPredicateFuncAdapter {
	return &IntPredicateFuncAdapter{call: call}
}
func NewPlainIntPredicateFuncAdapter(call func(int32) bool) *IntPredicateFuncAdapter {
	return NewIntPredicateFuncAdapter(func(_ *Execution, value int32) bool { return call(value) })
}
func NewIntPredicateSourceView(info *ObjectInfo, call func(*Execution, int32) bool) *IntPredicateFuncAdapter {
	return &IntPredicateFuncAdapter{ObjectInfo: info, call: call}
}
func (f *IntPredicateFuncAdapter) Test(value int32) bool {
	return f.TestJava2goExecution(NewExecution(), value)
}
func (f *IntPredicateFuncAdapter) TestJava2goExecution(e *Execution, value int32) bool {
	ReferenceRequireNonNull(f)
	return f.call(e, value)
}
func (f *IntPredicateFuncAdapter) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.IntPredicate"
}
func CallIntPredicateExecution(e *Execution, predicate IntPredicate, value int32) bool {
	ReferenceRequireNonNull(predicate)
	if aware, ok := predicate.(interface{ TestJava2goExecution(*Execution, int32) bool }); ok {
		return aware.TestJava2goExecution(e, value)
	}
	return predicate.Test(value)
}
func IntPredicateAndExecution(e *Execution, left, right IntPredicate) IntPredicate {
	ReferenceRequireNonNull(left)
	if source, ok := left.(interface {
		intPredicateAndOverride(*Execution, IntPredicate) (IntPredicate, bool)
	}); ok {
		if result, overridden := source.intPredicateAndOverride(e, right); overridden {
			return result
		}
	}
	ReferenceRequireNonNull(right)
	return NewIntPredicateFuncAdapter(func(caller *Execution, value int32) bool {
		return CallIntPredicateExecution(caller, left, value) && CallIntPredicateExecution(caller, right, value)
	})
}
func IntPredicateOrExecution(e *Execution, left, right IntPredicate) IntPredicate {
	ReferenceRequireNonNull(left)
	if source, ok := left.(interface {
		intPredicateOrOverride(*Execution, IntPredicate) (IntPredicate, bool)
	}); ok {
		if result, overridden := source.intPredicateOrOverride(e, right); overridden {
			return result
		}
	}
	ReferenceRequireNonNull(right)
	return NewIntPredicateFuncAdapter(func(caller *Execution, value int32) bool {
		return CallIntPredicateExecution(caller, left, value) || CallIntPredicateExecution(caller, right, value)
	})
}
func IntPredicateNegateExecution(e *Execution, predicate IntPredicate) IntPredicate {
	ReferenceRequireNonNull(predicate)
	if source, ok := predicate.(interface {
		intPredicateNegateOverride(*Execution) (IntPredicate, bool)
	}); ok {
		if result, overridden := source.intPredicateNegateOverride(e); overridden {
			return result
		}
	}
	return NewIntPredicateFuncAdapter(func(caller *Execution, value int32) bool { return !CallIntPredicateExecution(caller, predicate, value) })
}
func init() { RegisterJavaType("java.util.function.IntPredicate", ObjectTypeID) }
