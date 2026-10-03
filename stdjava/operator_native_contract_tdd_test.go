package stdjava

import "testing"

func TestUnaryOperatorParentReferenceContractTDD(t *testing.T) {
	e := NewExecution()
	unary := NewUnaryOperatorFuncAdapter(func(caller *Execution, s *JavaString) *JavaString {
		if caller != e {
			t.Fatal("Unary parent lost caller Execution")
		}
		return s
	})
	var parent Function[*JavaString, *JavaString] = unary
	if !ObjectInstanceOf(unary, "java.util.function.Function") {
		t.Fatal("UnaryOperator missing declared Function parent")
	}
	restored := ObjectView[Function[*JavaString, *JavaString]](any(unary), "java.util.function.Function")
	value := NewJavaStringUTF16([]uint16{115, 97, 109, 101})
	if !JavaReferenceEqual(unary, parent) || !JavaReferenceEqual(unary, restored) || CallFunctionExecution(e, restored, value) != value {
		t.Fatal("Unary parent alias lost identity or invocation")
	}
}

func TestBinaryOperatorParentReferenceContractTDD(t *testing.T) {
	e := NewExecution()
	value := NewJavaStringUTF16([]uint16{115, 97, 109, 101})
	binary := NewBinaryOperatorFuncAdapter(func(caller *Execution, a, b *JavaString) *JavaString {
		if caller != e {
			t.Fatal("Binary parent lost caller Execution")
		}
		return a
	})
	// This is the actual required parent Go ABI, independent of the production
	// type's name; nominal membership alone cannot satisfy it.
	type parentABI interface {
		Apply(*JavaString, *JavaString) *JavaString
	}
	var physical parentABI = binary
	if !ObjectInstanceOf(binary, "java.util.function.BiFunction") {
		t.Fatal("BinaryOperator missing declared BiFunction parent")
	}
	restoredBinary := ObjectView[parentABI](any(binary), "java.util.function.BiFunction")
	if !JavaReferenceEqual(binary, physical) || !JavaReferenceEqual(binary, restoredBinary) {
		t.Fatal("Binary parent view changed Java allocation")
	}
	if result := CallBinaryOperatorExecution(e, restoredBinary, value, NewJavaStringUTF16([]uint16{111, 116, 104, 101, 114})); result != value {
		t.Fatal("Binary parent callback identity lost")
	}
}

type operatorCollisionSource struct {
	*ObjectInfo
	want *Execution
}

func (x *operatorCollisionSource) ApplyAsInt(value int32) int32 { return -700 }
func (x *operatorCollisionSource) ApplyAsIntJava2goExecution1(e *Execution, value int32) int32 {
	if e != x.want {
		panic(NewIllegalStateException("wrong caller execution"))
	}
	return value + 1
}

type operatorUnarySourceView struct {
	*ObjectInfo
	source *operatorCollisionSource
}

func (v *operatorUnarySourceView) ApplyAsInt(a int32) int32 {
	return v.ApplyAsIntJava2goExecution(NewExecution(), a)
}
func (v *operatorUnarySourceView) ApplyAsIntJava2goExecution(e *Execution, a int32) int32 {
	return v.source.ApplyAsIntJava2goExecution1(e, a)
}

type operatorBinarySourceView struct {
	*ObjectInfo
	source *operatorCollisionSource
}

func (v *operatorBinarySourceView) ApplyAsInt(a, b int32) int32 {
	return v.ApplyAsIntJava2goExecution(NewExecution(), a, b)
}
func (v *operatorBinarySourceView) ApplyAsIntJava2goExecution(e *Execution, a, b int32) int32 {
	if e != v.source.want {
		panic(NewIllegalStateException("wrong binary execution"))
	}
	return a + b
}

func TestOperatorExplicitNativeViewPrecedesPublicShapeTDD(t *testing.T) {
	id := TypeID("atomic.control.OperatorCollision")
	RegisterJavaType(id, ObjectTypeID, "java.util.function.IntUnaryOperator", "java.util.function.IntBinaryOperator")
	e := NewExecution()
	source := &operatorCollisionSource{want: e}
	source.ObjectInfo = NewObjectInfo(id, func(requested TypeID) any {
		switch requested {
		case id:
			return source
		case "java.util.function.IntUnaryOperator":
			return &operatorUnarySourceView{ObjectInfo: source.ObjectInfo, source: source}
		case "java.util.function.IntBinaryOperator":
			return &operatorBinarySourceView{ObjectInfo: source.ObjectInfo, source: source}
		}
		return nil
	})
	unary := ObjectView[IntUnaryOperator](any(source), "java.util.function.IntUnaryOperator")
	binary := ObjectView[IntBinaryOperator](any(source), "java.util.function.IntBinaryOperator")
	if !JavaReferenceEqual(source, unary) || !JavaReferenceEqual(source, binary) || !JavaReferenceEqual(unary, binary) {
		t.Fatal("explicit SAM views must share source allocation")
	}
	if got := CallIntUnaryOperatorExecution(e, unary, 3); got != 4 {
		t.Fatalf("public structural shape bypassed collision-safe execution view: %d", got)
	}
	if got := CallIntBinaryOperatorExecution(e, binary, 3, 4); got != 7 {
		t.Fatalf("binary SAM view missing: %d", got)
	}
	if !ObjectInstanceOf(unary, id) || !ObjectInstanceOf(binary, id) {
		t.Fatal("views lost source dynamic type")
	}
}
