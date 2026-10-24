package transpiler

import (
	"fmt"
	"testing"
)

// RED-only public214 addition. Original assertions are retained unchanged.
func reflectionClassOrigin214Context(t *testing.T, source string) Ctx {
	t.Helper()
	h := setupParseHelper(t, source)
	owner := h.File.Symbols.FindClassScope("CarrierProbe")
	if owner == nil {
		t.Fatal("actual parsed CarrierProbe declaration missing")
	}
	ctx := classScopeCtx(owner, h.Ctx)
	if ctx.currentFile != h.File.Symbols || findFileScopeForClassScope(owner, ctx) != h.File.Symbols {
		t.Fatal("actual declaring file not retained")
	}
	return ctx
}

func reflectionClassOrigin214Formal(t *testing.T, ctx Ctx) inferredJavaTypeOrigin {
	t.Helper()
	methods := ctx.currentClass.FindMethod().ByOriginalName("take")
	if len(methods) != 1 || len(methods[0].Parameters) != 1 {
		t.Fatal("actual parsed take parameter missing")
	}
	ctx.localScope = methods[0]
	return declaredJavaTypeOrigin(definitionDeclaredJavaType(methods[0].Parameters[0]), ctx)
}

func TestReflectionClassDeclaredOriginsRetainParsedCarriers(t *testing.T) {
	for _, name := range []string{"Method", "Field", "Constructor"} {
		for _, form := range []string{"qualified", "explicit", "demand", "sameLeafBinder", "sourceHomonym", "valueRoot"} {
			t.Run(name+"/"+form, func(t *testing.T) {
				written, header, parameters, fields := "java.lang.reflect."+name, "", "", ""
				switch form {
				case "explicit":
					header, written = "import java.lang.reflect."+name+";", name
				case "demand":
					header, written = "import java.lang.reflect.*;", name
				case "sameLeafBinder":
					parameters = "<" + name + ">"
				case "sourceHomonym":
					header = "class " + name + "{}"
				case "valueRoot":
					fields = "int java;"
				}
				ctx := reflectionClassOrigin214Context(t, fmt.Sprintf("%sclass CarrierProbe%s{%svoid take(%s value){}}", header, parameters, fields, written))
				got := reflectionClassOrigin214Formal(t, ctx)
				t.Logf("REFLECT214_CARRIER case=%s/%s external=%q unresolved=%t nominal=%t parameter=%t ownerExact=%t", name, form, got.externalName, got.unresolved, got.nominalScope != nil, got.parameter != nil, got.declaringOwner == ctx.currentClass)
				if got.externalName != "java.lang.reflect."+name || got.unresolved || got.nominalScope != nil || got.parameter != nil || got.declaringOwner != ctx.currentClass {
					t.Fatalf("canonical parsed reflection class lost declaration identity: %+v", got)
				}
				_, exact, applies, handled := declaredOriginConversion(got, got, ctx)
				if !handled || !exact || !applies {
					t.Fatal("same actual declared class failed exact argument conversion")
				}
			})
		}
	}
}

func TestReflectionClassDeclaredOriginsKeepOwnershipShields(t *testing.T) {
	for _, name := range []string{"Method", "Field", "Constructor"} {
		for _, kind := range []string{"sourceLeaf", "classBinder", "methodBinder", "sourceRoot", "foreignImport", "unknownDemand"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				source := ""
				switch kind {
				case "sourceLeaf":
					source = fmt.Sprintf("class %s{}class CarrierProbe{void take(%s value){}}", name, name)
				case "classBinder":
					source = fmt.Sprintf("class CarrierProbe<%s>{void take(%s value){}}", name, name)
				case "methodBinder":
					source = fmt.Sprintf("class CarrierProbe{<%s>void take(%s value){}}", name, name)
				case "sourceRoot":
					source = fmt.Sprintf("class CarrierProbe{static class java{static class lang{static class reflect{static class %s{}}}}void take(java.lang.reflect.%s value){}}", name, name)
				case "foreignImport":
					source = fmt.Sprintf("import foreign.%s;class CarrierProbe{void take(%s value){}}", name, name)
				case "unknownDemand":
					source = fmt.Sprintf("import foreign.*;import java.lang.reflect.*;class CarrierProbe{void take(%s value){}}", name)
				}
				ctx := reflectionClassOrigin214Context(t, source)
				got := reflectionClassOrigin214Formal(t, ctx)
				t.Logf("REFLECT214_SHIELD case=%s/%s external=%q unresolved=%t nominal=%t parameter=%t", name, kind, got.externalName, got.unresolved, got.nominalScope != nil, got.parameter != nil)
				if got.externalName != "" {
					t.Fatalf("source/binder/foreign declaration borrowed native class identity: %+v", got)
				}
				if kind == "classBinder" && got.parameter != ctx.currentClass.TypeParameters[0].Declaration {
					t.Fatal("actual class binder identity lost")
				}
				if kind == "methodBinder" && got.parameter != ctx.currentClass.FindMethod().ByOriginalName("take")[0].TypeParameters[0].Declaration {
					t.Fatal("actual method binder identity lost")
				}
			})
		}
		ctx := reflectionClassOrigin214Context(t, "class CarrierProbe{}")
		noFile, noOwner, stale := ctx.Clone(), ctx.Clone(), ctx.Clone()
		noFile.currentFile, noOwner.currentClass = nil, nil
		copyFile := *ctx.currentFile
		stale.currentFile = &copyFile
		for index, bad := range []Ctx{noFile, noOwner, stale} {
			got := declaredExternalNominalOwner("java.lang.reflect."+name, nil, bad)
			t.Logf("REFLECT214_CONTEXT case=%s/%s owner=%q", name, []string{"missingFile", "missingOwner", "staleDetachedFile"}[index], got)
			if got != "" {
				t.Fatalf("native class acquired without its actual declaring file: %s", got)
			}
		}
	}
}

func TestReflectionClassMethodArgumentRetainsPrimitiveResult(t *testing.T) {
	source := `class CarrierProbe{static class Scope<Method>{boolean check(java.lang.reflect.Method value){return value.isVarArgs();}}static boolean run(java.lang.reflect.Method value){return new Scope<Integer>().check(value);}}`
	ctx := reflectionClassOrigin214Context(t, source)
	methods := ctx.currentClass.FindMethod().ByOriginalName("run")
	if len(methods) != 1 {
		t.Fatal("actual parsed run declaration missing")
	}
	ctx.localScope = methods[0]
	call := findNode(methods[0].DeclarationNode, "method_invocation")
	if call == nil || call.ChildByFieldName("name").Content([]byte(source)) != "check" {
		t.Fatal("actual source invocation missing")
	}
	result, known := inferUserMethodReturnType(call, ctx, []byte(source))
	t.Logf("REFLECT214_PRIMITIVE type=%q known=%t", result, known)
	if !known || result != "boolean" {
		t.Fatalf("declaration-owned Method argument erased primitive source result: type=%q known=%v", result, known)
	}
}

// Guard contexts deliberately include unmodeled import owners. They test that
// an unresolved competing TYPE namespace cannot donate canonical class identity;
// they make no assertion that javac accepts an unavailable imported owner.
func TestReflectionClassDeclaredOriginsCompetingStaticTypeImports(t *testing.T) {
	for _, name := range []string{"Method", "Field", "Constructor"} {
		for _, form := range []string{"demand", "explicit", "qualified"} {
			for _, competing := range []string{"relevantSingle", "unknownWildcard", "unrelatedSingle", "knownAbsentWildcard"} {
				t.Run(name+"/"+form+"/"+competing, func(t *testing.T) {
					written, header := name, "import java.lang.reflect.*;"
					switch form {
					case "explicit":
						header = "import java.lang.reflect." + name + ";"
					case "qualified":
						written, header = "java.lang.reflect."+name, ""
					}
					first := name
					if form == "qualified" {
						first = "java"
					}
					staticImport, want := "", false
					switch competing {
					case "relevantSingle":
						staticImport = "import static foreign.Owner." + first + ";"
					case "unknownWildcard":
						staticImport = "import static foreign.Owner.*;"
					case "unrelatedSingle":
						staticImport, want = "import static foreign.Owner.other;", true
					case "knownAbsentWildcard":
						staticImport, want = "import static java.util.Map.*;", true
					}
					ctx := reflectionClassOrigin214Context(t, staticImport+header+"class CarrierProbe{void take("+written+" value){}}")
					got := reflectionClassOrigin214Formal(t, ctx)
					t.Logf("REFLECT214_STATIC case=%s/%s/%s want=%t external=%q unresolved=%t nominal=%t parameter=%t", name, form, competing, want, got.externalName, got.unresolved, got.nominalScope != nil, got.parameter != nil)
					if want {
						if got.unresolved || got.externalName != "java.lang.reflect."+name || got.parameter != nil || got.nominalScope != nil {
							t.Fatalf("irrelevant or certified-absent static TYPE import stole canonical declaration: %+v", got)
						}
					} else if got.externalName != "" {
						t.Fatalf("unresolved relevant static TYPE namespace donated canonical class: %+v", got)
					}
				})
			}
		}
	}
}

// A certified unrelated namespace cannot erase another unresolved TYPE import.
// These guard-only contexts do not claim javac accepts an unavailable owner.
func TestReflectionClassDeclaredOriginsMixedStaticTypeImports(t *testing.T) {
	for _, name := range []string{"Method", "Field", "Constructor"} {
		for _, kind := range []string{"relevantSingle", "unknownWildcard", "unrelatedSingle"} {
			for _, order := range []string{"certifiedFirst", "unknownFirst"} {
				t.Run(name+"/"+kind+"/"+order, func(t *testing.T) {
					unknown := "import static foreign.Owner." + name + ";"
					want := false
					switch kind {
					case "unknownWildcard":
						unknown = "import static foreign.Owner.*;"
					case "unrelatedSingle":
						unknown, want = "import static foreign.Owner.other;", true
					}
					imports := "import static java.util.Map.*;" + unknown
					if order == "unknownFirst" {
						imports = unknown + "import static java.util.Map.*;"
					}
					source := "import java.lang.reflect." + name + ";" + imports + "class CarrierProbe{void take(" + name + " value){}}"
					ctx := reflectionClassOrigin214Context(t, source)
					got := reflectionClassOrigin214Formal(t, ctx)
					if want {
						if got.unresolved || got.externalName != "java.lang.reflect."+name || got.parameter != nil || got.nominalScope != nil {
							t.Fatalf("unrelated single import stole canonical reflection class: %+v", got)
						}
					} else if got.externalName != "" {
						t.Fatalf("certified namespace hid unresolved competing TYPE import: %+v", got)
					}
				})
			}
		}
	}
}
