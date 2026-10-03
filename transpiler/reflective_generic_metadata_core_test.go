package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"testing"
)

// Source and metadata expectations come from the independent generic control:
// d002d48b, raw seed17/41/97 each repeated3. Physical Go type arguments must
// never replace these Java declaration/type identities.
func TestReflectiveGenericMetadataCoreAST(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"probe/app/Main.java":    "package probe.app;\nimport probe.model.*;\nimport java.lang.reflect.*;\nimport java.util.*;\npublic final class Main {\n public static void main(String[] args)throws Exception{\n  int seed=Integer.parseInt(args[0]);Order state=new Order();state.state=\"seed-\"+seed;\n  Base<List<Order>> anonymous=new Base<List<Order>>(){};anonymous.state=new ArrayList<>();anonymous.state.add(state);\n  ParameterizedType inherited=(ParameterizedType)Order.class.getGenericSuperclass();\n  ParameterizedType captured=(ParameterizedType)anonymous.getClass().getGenericSuperclass();\n  ParameterizedType list=(ParameterizedType)captured.getActualTypeArguments()[0];\n  System.out.println(\"inherited|\"+inherited.getTypeName()+\"|\"+(inherited.getRawType()==Base.class)+\"|\"+(inherited.getActualTypeArguments()[0]==String.class)+\"|\"+(inherited.getOwnerType()==null));\n  System.out.println(\"anonymous|\"+captured.getTypeName()+\"|\"+(captured.getRawType()==Base.class)+\"|\"+(list.getRawType()==List.class)+\"|\"+(list.getActualTypeArguments()[0]==Order.class));\n  for(Field field:Order.class.getDeclaredFields())System.out.println(\"field|\"+field.getName()+\"|\"+field.getGenericType().getTypeName());\n  TypeVariable<?> variable=Base.class.getTypeParameters()[0];\n  System.out.println(\"variable|\"+variable.getName()+\"|\"+(variable.getGenericDeclaration()==Base.class)+\"|\"+(variable.getBounds()[0]==Object.class));\n  System.out.println(\"ordinary|\"+(Object.class.getGenericSuperclass()==null)+\"|\"+(String.class.getGenericSuperclass()==Object.class));\n  anonymous.state.get(0).state+=\"-mutated\";\n  System.out.println(\"state|\"+state.state+\"|\"+anonymous.state.size());\n }\n}\n",
		"probe/model/Base.java":  "package probe.model;\npublic class Base<T> { public T state; }\n",
		"probe/model/Order.java": "package probe.model;\nimport java.util.List;\nimport java.util.Map;\npublic final class Order extends Base<String> { public List<String> lines; public Map<String,Integer> metadata; }\n",
	})
	metadata := func(t *testing.T, name string) *ast.CompositeLit {
		t.Helper()
		scope := findQualifiedSourceClass(name)
		if scope == nil {
			t.Fatal("missing source class", name)
		}
		stmt := sourceClassMetadataStmt(scope, classScopeCtx(scope, ctx))
		if stmt == nil {
			t.Fatal("missing class metadata", name)
		}
		expression, ok := stmt.(*ast.ExprStmt)
		if !ok {
			t.Fatal("metadata is not expression")
		}
		call, ok := expression.X.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			t.Fatal("metadata registration shape")
		}
		return reflectCoreComposite(t, call.Args[0])
	}
	t.Run("source_class_identity_control", func(t *testing.T) {
		for _, name := range []string{"probe.model.Base", "probe.model.Order"} {
			scope := findQualifiedSourceClass(name)
			if len(scope.OwnTypeParameters()) != map[string]int{"probe.model.Base": 1, "probe.model.Order": 0}[name] {
				t.Fatal("source declaration arity")
			}
			got := reflectCoreTypeID(t, reflectCoreKey(t, metadata(t, name), "Type"))
			if got != name {
				t.Errorf("class declaration identity=%q want%q", got, name)
			}
		}
	})
	t.Run("class_variable_bounds", func(t *testing.T) {
		parameters := reflectCoreSlice(t, reflectCoreKey(t, metadata(t, "probe.model.Base"), "TypeParameters"))
		if len(parameters) != 1 {
			t.Fatalf("declared parameters=%d want1", len(parameters))
		}
		variable := reflectCoreComposite(t, parameters[0])
		if reflectCoreString(t, reflectCoreKey(t, variable, "Name")) != "T" {
			t.Fatal("source variable name")
		}
		bounds := reflectCoreSlice(t, reflectCoreKey(t, variable, "Bounds"))
		if len(bounds) != 1 {
			t.Fatalf("bounds=%d want1", len(bounds))
		}
		reflectCoreClass(t, bounds[0], "java.lang.Object")
	})
	t.Run("concrete_generic_superclass", func(t *testing.T) {
		tree := reflectCoreKey(t, metadata(t, "probe.model.Order"), "GenericSuperclass")
		reflectCoreParameterized(t, tree, "probe.model.Base", []string{"java.lang.String"})
	})
	for _, test := range []struct {
		name, raw string
		arguments []string
	}{
		{"lines", "java.util.List", []string{"java.lang.String"}},
		{"metadata", "java.util.Map", []string{"java.lang.String", "java.lang.Integer"}},
	} {
		t.Run("field_"+test.name, func(t *testing.T) {
			fields := reflectCoreSlice(t, reflectCoreKey(t, metadata(t, "probe.model.Order"), "Fields"))
			for _, field := range fields {
				descriptor := reflectCoreComposite(t, field)
				if reflectCoreString(t, reflectCoreKey(t, descriptor, "Name")) == test.name {
					reflectCoreParameterized(t, reflectCoreKey(t, descriptor, "GenericType"), test.raw, test.arguments)
					return
				}
			}
			t.Fatal("missing declared field", test.name)
		})
	}
}

func reflectCoreComposite(t *testing.T, e ast.Expr) *ast.CompositeLit {
	t.Helper()
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	c, ok := e.(*ast.CompositeLit)
	if !ok {
		t.Fatalf("descriptor is %T", e)
	}
	return c
}
func reflectCoreOptionalKey(c *ast.CompositeLit, name string) ast.Expr {
	for _, e := range c.Elts {
		if p, ok := e.(*ast.KeyValueExpr); ok {
			if k, ok := p.Key.(*ast.Ident); ok && k.Name == name {
				return p.Value
			}
		}
	}
	return nil
}
func reflectCoreKey(t *testing.T, c *ast.CompositeLit, name string) ast.Expr {
	t.Helper()
	e := reflectCoreOptionalKey(c, name)
	if e == nil {
		t.Fatalf("missing Java generic descriptor key %s", name)
	}
	return e
}
func reflectCoreSlice(t *testing.T, e ast.Expr) []ast.Expr {
	t.Helper()
	return reflectCoreComposite(t, e).Elts
}
func reflectCoreString(t *testing.T, e ast.Expr) string {
	t.Helper()
	l, ok := e.(*ast.BasicLit)
	if !ok || l.Kind != token.STRING {
		t.Fatalf("source name is %T", e)
	}
	v, err := strconv.Unquote(l.Value)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func reflectCoreTypeID(t *testing.T, e ast.Expr) string {
	t.Helper()
	if c, ok := e.(*ast.CallExpr); ok && len(c.Args) == 1 {
		return reflectCoreString(t, c.Args[0])
	}
	if s, ok := e.(*ast.SelectorExpr); ok {
		switch s.Sel.Name {
		case "ObjectTypeID":
			return "java.lang.Object"
		case "StringTypeID":
			return "java.lang.String"
		case "IntegerTypeID":
			return "java.lang.Integer"
		}
	}
	t.Fatalf("unrecognized exact TypeID expression %T", e)
	return ""
}
func reflectCoreKind(t *testing.T, c *ast.CompositeLit, want string) {
	t.Helper()
	e := reflectCoreKey(t, c, "Kind")
	s, ok := e.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != want {
		t.Fatalf("reflective kind %T want%s", e, want)
	}
}
func reflectCoreClass(t *testing.T, e ast.Expr, want string) {
	t.Helper()
	c := reflectCoreComposite(t, e)
	reflectCoreKind(t, c, "ReflectClassKind")
	if got := reflectCoreTypeID(t, reflectCoreKey(t, c, "Raw")); got != want {
		t.Errorf("raw=%q want%q", got, want)
	}
}
func reflectCoreParameterized(t *testing.T, e ast.Expr, raw string, want []string) {
	t.Helper()
	c := reflectCoreComposite(t, e)
	reflectCoreKind(t, c, "ReflectParameterizedKind")
	if got := reflectCoreTypeID(t, reflectCoreKey(t, c, "Raw")); got != raw {
		t.Errorf("raw=%q want%q", got, raw)
	}
	if owner := reflectCoreOptionalKey(c, "Owner"); owner != nil {
		if n, ok := owner.(*ast.Ident); !ok || n.Name != "nil" {
			t.Error("top-level type has nonnull owner")
		}
	}
	arguments := reflectCoreSlice(t, reflectCoreKey(t, c, "Arguments"))
	if len(arguments) != len(want) {
		t.Fatalf("args=%d want%d", len(arguments), len(want))
	}
	for i, v := range want {
		reflectCoreClass(t, arguments[i], v)
	}
}
