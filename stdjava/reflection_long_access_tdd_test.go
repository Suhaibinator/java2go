package stdjava

import "testing"

type reflectionLongAccessTDD interface {
	GetLongExecution(*Execution, any) int64
	SetLongExecution(*Execution, any, int64)
}
type reflectionLongFixtureTDD struct {
	Byte      int8
	Short     int16
	Char      rune
	Int       int32
	Long      int64
	Float     float32
	Double    float64
	Reference any
	Hidden    int64
	Fixed     int64
	Cell      VolatileFieldCell
}

func (*reflectionLongFixtureTDD) JavaDynamicTypeID() TypeID { return "test.TypedLongFieldFixture" }
func TestReflectionLongPrimitiveAccessTDD(t *testing.T) {
	kinds := []struct {
		name, goName string
		id           TypeID
	}{
		{"byte", "Byte", PrimitiveByteTypeID}, {"short", "Short", PrimitiveShortTypeID}, {"char", "Char", PrimitiveCharTypeID}, {"int", "Int", PrimitiveIntTypeID}, {"long", "Long", PrimitiveLongTypeID}, {"float", "Float", PrimitiveFloatTypeID}, {"double", "Double", PrimitiveDoubleTypeID}, {"reference", "Reference", ObjectTypeID},
	}
	const id TypeID = "test.TypedLongFieldFixture"
	RegisterJavaType(id, ObjectTypeID)
	fields := []FieldDescriptor{}
	for _, k := range kinds {
		fields = append(fields, FieldDescriptor{Name: k.name, GoName: k.goName, Type: k.id})
	}
	fields = append(fields, FieldDescriptor{Name: "hidden", GoName: "Hidden", Type: PrimitiveLongTypeID, NonPublic: true}, FieldDescriptor{Name: "fixed", GoName: "Fixed", Type: PrimitiveLongTypeID, Final: true})
	fields = append(fields, FieldDescriptor{Name: "cell", Type: PrimitiveLongTypeID, Get: func(e *Execution, v any) any { return VolatileLoad[int64](&v.(*reflectionLongFixtureTDD).Cell) }, Set: func(e *Execution, v, number any) { VolatileStore(&v.(*reflectionLongFixtureTDD).Cell, number.(int64)) }})
	observed := []*Execution{}
	initialized := []*Execution{}
	fields = append(fields, FieldDescriptor{Name: "static", Type: PrimitiveLongTypeID, StaticGet: func(e *Execution) any { observed = append(observed, e); return int64(17) }, StaticSet: func(e *Execution, v any) { observed = append(observed, e) }})
	RegisterClassDescriptor(ClassDescriptor{Type: id, Modifiers: 1, HasModifiers: true, Initialize: func(e *Execution) { initialized = append(initialized, e) }, Fields: fields})
	class := ClassLiteral(id)
	access := func(t *testing.T, name string) reflectionLongAccessTDD {
		t.Helper()
		f := class.GetDeclaredField(name)
		api, ok := any(f).(reflectionLongAccessTDD)
		if !ok {
			t.Fatal("Field lacks typed long Execution API")
		}
		return api
	}
	fixture := func() *reflectionLongFixtureTDD {
		return &reflectionLongFixtureTDD{Byte: -3, Short: -5, Char: 65535, Int: -7, Long: 9007199254740993, Float: 2, Double: 3, Reference: BoxLong(4)}
	}
	values := map[string]int64{"byte": -3, "short": -5, "char": 65535, "int": -7, "long": 9007199254740993}
	for _, k := range kinds {
		t.Run("get_"+k.name, func(t *testing.T) {
			api := access(t, k.name)
			if want, ok := values[k.name]; ok {
				if got := api.GetLongExecution(NewExecution(), fixture()); got != want {
					t.Fatalf("getLong=%d want%d", got, want)
				}
			} else {
				expectBoxedException(t, "IllegalArgumentException", func() { api.GetLongExecution(NewExecution(), fixture()) })
			}
		})
	}
	for _, name := range []string{"long", "float", "double", "int", "reference"} {
		t.Run("set_"+name, func(t *testing.T) {
			api := access(t, name)
			v := fixture()
			if name == "int" || name == "reference" {
				expectBoxedException(t, "IllegalArgumentException", func() { api.SetLongExecution(NewExecution(), v, 19) })
				if v.Int != -7 || UnboxLong(v.Reference.(*Long)) != 4 {
					t.Fatal("rejected write mutated storage")
				}
				return
			}
			api.SetLongExecution(NewExecution(), v, 19)
			if name == "long" && v.Long != 19 || name == "float" && v.Float != 19 || name == "double" && v.Double != 19 {
				t.Fatal("setLong failed widening")
			}
		})
	}
	t.Run("volatile_same_cell", func(t *testing.T) {
		api := access(t, "cell")
		v := fixture()
		VolatileStore(&v.Cell, int64(23))
		if api.GetLongExecution(NewExecution(), v) != 23 {
			t.Fatal("getLong bypassed cell")
		}
		api.SetLongExecution(NewExecution(), v, 29)
		if VolatileLoad[int64](&v.Cell) != 29 {
			t.Fatal("setLong replaced or bypassed cell")
		}
	})
	t.Run("execution_context", func(t *testing.T) {
		api := access(t, "static")
		observed = nil
		initialized = nil
		e := NewExecution()
		api.SetLongExecution(e, nil, 31)
		if got := api.GetLongExecution(e, nil); got != 17 {
			t.Fatalf("typed getter returned %d, want callback value 17", got)
		}
		if len(observed) != 2 {
			t.Fatalf("typed setter/getter callback count = %d, want 2", len(observed))
		}
		if observed[0] != e {
			t.Fatal("typed setter callback lost exact caller Execution")
		}
		if observed[1] != e {
			t.Fatal("typed getter callback lost exact caller Execution")
		}
		if len(initialized) != 1 {
			t.Fatalf("declaring class initializer count = %d, want exactly once", len(initialized))
		}
		if initialized[0] != e {
			t.Fatal("class initializer lost the first active caller Execution")
		}
	})
	t.Run("private_access", func(t *testing.T) {
		api := access(t, "hidden")
		v := fixture()
		expectBoxedException(t, "IllegalAccessException", func() { api.SetLongExecution(NewExecution(), v, 37) })
		expectBoxedException(t, "IllegalAccessException", func() { api.GetLongExecution(NewExecution(), v) })
		f := class.GetDeclaredField("hidden")
		f.SetAccessible(true)
		api = any(f).(reflectionLongAccessTDD)
		api.SetLongExecution(NewExecution(), v, 37)
		if api.GetLongExecution(NewExecution(), v) != 37 {
			t.Fatal("access override failed")
		}
	})
	t.Run("null_receiver", func(t *testing.T) {
		api := access(t, "long")
		expectBoxedException(t, "NullPointerException", func() { api.GetLongExecution(NewExecution(), nil) })
		expectBoxedException(t, "NullPointerException", func() { api.SetLongExecution(NewExecution(), nil, 1) })
	})
	t.Run("wrong_receiver", func(t *testing.T) {
		api := access(t, "long")
		expectBoxedException(t, "IllegalArgumentException", func() { api.GetLongExecution(NewExecution(), "wrong") })
		expectBoxedException(t, "IllegalArgumentException", func() { api.SetLongExecution(NewExecution(), "wrong", 1) })
	})
	t.Run("final_write", func(t *testing.T) {
		api := access(t, "fixed")
		v := fixture()
		expectBoxedException(t, "IllegalAccessException", func() { api.SetLongExecution(NewExecution(), v, 41) })
		if v.Fixed != 0 {
			t.Fatal("final rejection mutated storage")
		}
	})
	t.Run("static_init_before_final", func(t *testing.T) {
		const finalID TypeID = "test.TypedLongFinalFixture"
		RegisterJavaType(finalID, ObjectTypeID)
		count := 0
		e := NewExecution()
		RegisterClassDescriptor(ClassDescriptor{Type: finalID, Modifiers: 1, HasModifiers: true, Initialize: func(got *Execution) {
			if got != e {
				t.Fatal("init lost Execution")
			}
			count++
		}, Fields: []FieldDescriptor{{Name: "fixed", Type: PrimitiveLongTypeID, Final: true, StaticGet: func(*Execution) any { return int64(0) }}}})
		f := ClassLiteral(finalID).GetDeclaredField("fixed")
		api, ok := any(f).(reflectionLongAccessTDD)
		if !ok {
			t.Fatal("Field lacks typed long Execution API")
		}
		expectBoxedException(t, "IllegalAccessException", func() { api.SetLongExecution(e, nil, 43) })
		if count != 1 {
			t.Fatal("typed final static setter bypassed initialization")
		}
	})
}
