package transpiler

import (
	"go/ast"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file registers the java.util collection intrinsics: List (ArrayList /
// LinkedList), Map (HashMap / TreeMap), Set (HashSet / TreeSet), Optional, and
// the Collections / Arrays static utilities. They are mapped onto the slice- and
// map-backed runtime types in stdjava (list.go, map.go, set.go, optional.go,
// collections_common.go).
//
// The List/Map/Set interface names and their concrete implementations all map to
// the same stdjava type, so an instance intrinsic is registered under every name
// a receiver might carry (e.g. a variable declared `List<T>` but assigned an
// ArrayList reports type List; one declared `ArrayList<T>` reports ArrayList).

func init() {
	registerCollectionConstructors()
	registerListIntrinsics()
	registerMapIntrinsics()
	registerSetIntrinsics()
	registerOptionalIntrinsics()
	registerCollectionsStatics()
}

// listTypeNames are the Java types that a List-valued receiver may be declared
// as. mapTypeNames and setTypeNames are the equivalents for maps and sets.
var (
	listTypeNames = []string{"List", "ArrayList", "LinkedList", "AbstractList"}
	mapTypeNames  = []string{"Map", "HashMap", "TreeMap", "LinkedHashMap", "AbstractMap"}
	setTypeNames  = []string{"Set", "HashSet", "TreeSet", "LinkedHashSet", "AbstractSet"}
)

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// optionalElementTypeExpr returns the Go type expression for T when the expected
// type in scope is an Optional<T>, or nil if it cannot be determined. Used to
// supply Optional.empty()'s type argument explicitly.
func optionalElementTypeExpr(ctx Ctx) ast.Expr {
	expected := strings.TrimSpace(ctx.expectedType)
	if expected == "" {
		return nil
	}
	base, typeArgs := parseJavaTypeString(expected)
	if stripJavaQualifier(base) != "Optional" || len(typeArgs) != 1 {
		return nil
	}
	return javaTypeStringToGoTypeExpr(typeArgs[0], inScopeTypeParameters(ctx), ctx)
}

// collectionNeedsSliceForRange reports whether an enhanced-for over the given
// expression must range over its stdjava .Slice() view rather than the value
// directly. This is true for List and Set receivers (pointer types backed by a
// slice). Map iteration in Java goes through keySet/values/entrySet, which the
// intrinsics already lower to slices, so maps are not included.
func collectionNeedsSliceForRange(node *sitter.Node, ctx Ctx, source []byte) bool {
	javaType, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		return false
	}
	base, _ := parseJavaTypeString(javaType)
	name := stripJavaQualifier(base)
	return containsString(listTypeNames, name) || containsString(setTypeNames, name)
}

// collectionTypeExpr maps a Java collection type name plus its type-argument
// strings onto the corresponding stdjava Go type expression, or nil if the name
// is not a collection type. List/Map/Set are reference types and map to a
// pointer (mutations are shared); Optional is a value type.
func collectionTypeExpr(baseName string, typeArgs, scopeTypeParams []string, ctx Ctx) ast.Expr {
	argExprs := func() []ast.Expr {
		exprs := make([]ast.Expr, 0, len(typeArgs))
		for _, ta := range typeArgs {
			exprs = append(exprs, javaTypeStringToGoTypeExpr(ta, scopeTypeParams, ctx))
		}
		return exprs
	}

	switch {
	case containsString(listTypeNames, baseName):
		return &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("List", ctx), argExprs())}
	case containsString(mapTypeNames, baseName):
		return &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("Map", ctx), argExprs())}
	case containsString(setTypeNames, baseName):
		return &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("Set", ctx), argExprs())}
	case baseName == "Optional":
		return applyTypeArguments(stdjavaQualifiedExpr("Optional", ctx), argExprs())
	}
	return nil
}

func registerCollectionConstructors() {
	// new ArrayList<T>() / new LinkedList<T>() -> stdjava.NewList[T]()
	for _, name := range []string{"ArrayList", "LinkedList"} {
		registerConstructorIntrinsic(name, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			// Only the no-arg constructor is mapped; copy-constructors fall through.
			if len(args) != 0 {
				return nil
			}
			return stdjavaGenericCall(ctx, "NewList", typeArgs, nil)
		})
	}
	// new HashMap<K,V>() / new TreeMap<K,V>() -> stdjava.NewMap[K,V]()
	for _, name := range []string{"HashMap", "TreeMap", "LinkedHashMap"} {
		registerConstructorIntrinsic(name, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaGenericCall(ctx, "NewMap", typeArgs, nil)
		})
	}
	// new HashSet<T>() / new TreeSet<T>() -> stdjava.NewSet[T]()
	for _, name := range []string{"HashSet", "TreeSet", "LinkedHashSet"} {
		registerConstructorIntrinsic(name, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaGenericCall(ctx, "NewSet", typeArgs, nil)
		})
	}
}

// registerForTypes registers the same instance-method generator under each of
// the given Java receiver type names.
func registerForTypes(typeNames []string, method string, gen intrinsicGenerator) {
	for _, t := range typeNames {
		registerInstanceIntrinsic(t, method, gen)
	}
}

func registerListIntrinsics() {
	method := func(goName string, argc int) intrinsicGenerator {
		return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, argc) {
				return nil
			}
			return methodCall(recv, goName, args...)
		}
	}
	registerForTypes(listTypeNames, "add", method("Add", 1))
	registerForTypes(listTypeNames, "get", method("Get", 1))
	registerForTypes(listTypeNames, "set", method("Set", 2))
	registerForTypes(listTypeNames, "size", method("Size", 0))
	registerForTypes(listTypeNames, "isEmpty", method("IsEmpty", 0))
	registerForTypes(listTypeNames, "clear", method("Clear", 0))
	registerForTypes(listTypeNames, "contains", method("Contains", 1))
	registerForTypes(listTypeNames, "indexOf", method("IndexOf", 1))
	registerForTypes(listTypeNames, "addAll", method("AddAll", 1))
	registerForTypes(listTypeNames, "toArray", method("ToArray", 0))
	// remove(int) maps to RemoveAt; remove(Object) is a different overload that is
	// not handled here (it would need element-type analysis to disambiguate).
	registerForTypes(listTypeNames, "remove", method("RemoveAt", 1))
}

func registerMapIntrinsics() {
	method := func(goName string, argc int) intrinsicGenerator {
		return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, argc) {
				return nil
			}
			return methodCall(recv, goName, args...)
		}
	}
	registerForTypes(mapTypeNames, "put", method("Put", 2))
	registerForTypes(mapTypeNames, "get", method("Get", 1))
	registerForTypes(mapTypeNames, "getOrDefault", method("GetOrDefault", 2))
	registerForTypes(mapTypeNames, "containsKey", method("ContainsKey", 1))
	registerForTypes(mapTypeNames, "containsValue", method("ContainsValue", 1))
	registerForTypes(mapTypeNames, "remove", method("Remove", 1))
	registerForTypes(mapTypeNames, "size", method("Size", 0))
	registerForTypes(mapTypeNames, "isEmpty", method("IsEmpty", 0))
	registerForTypes(mapTypeNames, "clear", method("Clear", 0))
	registerForTypes(mapTypeNames, "keySet", method("KeySet", 0))
	registerForTypes(mapTypeNames, "values", method("Values", 0))
	registerForTypes(mapTypeNames, "entrySet", method("EntrySet", 0))
}

func registerSetIntrinsics() {
	method := func(goName string, argc int) intrinsicGenerator {
		return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, argc) {
				return nil
			}
			return methodCall(recv, goName, args...)
		}
	}
	registerForTypes(setTypeNames, "add", method("Add", 1))
	registerForTypes(setTypeNames, "contains", method("Contains", 1))
	registerForTypes(setTypeNames, "remove", method("Remove", 1))
	registerForTypes(setTypeNames, "size", method("Size", 0))
	registerForTypes(setTypeNames, "isEmpty", method("IsEmpty", 0))
	registerForTypes(setTypeNames, "clear", method("Clear", 0))
}

func registerOptionalIntrinsics() {
	// Optional.of/empty/ofNullable are static factories.
	registerStaticIntrinsic("Optional", "of", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		// When the expected type is Optional<T>, instantiate explicitly so the
		// element type matches Java (e.g. Optional<Integer> -> int32, not the Go
		// `int` that a bare integer literal would infer).
		if elem := optionalElementTypeExpr(ctx); elem != nil {
			return stdjavaGenericCall(ctx, "OptionalOf", []ast.Expr{elem}, []ast.Expr{args[0]})
		}
		return stdjavaCall(ctx, "OptionalOf", args[0])
	})
	registerStaticIntrinsic("Optional", "empty", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		// Go cannot infer OptionalEmpty's type parameter from the call's context
		// (return position / assignment), so supply it explicitly from the
		// expected Optional<T> type when known.
		if elem := optionalElementTypeExpr(ctx); elem != nil {
			return stdjavaGenericCall(ctx, "OptionalEmpty", []ast.Expr{elem}, nil)
		}
		return stdjavaCall(ctx, "OptionalEmpty")
	})

	// Instance methods on an Optional receiver.
	registerInstanceIntrinsic("Optional", "isPresent", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return methodCall(recv, "IsPresent")
	})
	registerInstanceIntrinsic("Optional", "isEmpty", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return methodCall(recv, "IsEmpty")
	})
	registerInstanceIntrinsic("Optional", "get", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return methodCall(recv, "Get")
	})
	registerInstanceIntrinsic("Optional", "orElse", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return methodCall(recv, "OrElse", args[0])
	})
	registerLambdaShape("Optional", "ifPresent", lambdaResultVoid)
	registerLambdaShape("Optional", "map", lambdaResultInferred)
	registerInstanceIntrinsic("Optional", "ifPresent", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return methodCall(recv, "IfPresent", args[0])
	})
	// map introduces a new result type parameter, which a Go method cannot, so it
	// is a free function: stdjava.OptionalMap(o, mapper).
	registerInstanceIntrinsic("Optional", "map", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "OptionalMap", recv, args[0])
	})
}

func registerCollectionsStatics() {
	// java.util.Collections
	registerStaticIntrinsic("Collections", "sort", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "SortOrdered", args[0])
	})
	registerStaticIntrinsic("Collections", "reverse", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ReverseList", args[0])
	})
	registerStaticIntrinsic("Collections", "max", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "MaxOrdered", args[0])
	})
	registerStaticIntrinsic("Collections", "min", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "MinOrdered", args[0])
	})
	registerStaticIntrinsic("Collections", "emptyList", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "EmptyList")
	})
	registerStaticIntrinsic("Collections", "singletonList", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "SingletonList", args[0])
	})
	registerStaticIntrinsic("Collections", "unmodifiableList", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "UnmodifiableList", args[0])
	})

	// java.util.Arrays
	registerStaticIntrinsic("Arrays", "asList", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return stdjavaCall(ctx, "AsList", args...)
	})
	registerStaticIntrinsic("Arrays", "sort", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "SortArray", args[0])
	})
	registerStaticIntrinsic("Arrays", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ArrayToString", args[0])
	})
	registerStaticIntrinsic("Arrays", "deepToString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ArrayDeepToString", args[0])
	})
}
