package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

const declaredOriginRefusalPositiveJava = `package p;
public class Use<Map> {
 static int marks;
 static void takeNested(java.util.Map<java.util.List<?>,java.lang.reflect.Type> value){marks+=value.size();}
 static void takeWildcard(java.util.Map<? extends Number,? extends CharSequence> value){marks+=value.size();}
 static void takeMap(java.util.Map<String,Integer> value){marks+=value.size();}
 static void takeObject(Object value){if(value!=null)marks++;}
 static <E> void takeE(E value){if(value!=null)marks++;}
 static void nested(java.util.HashMap<java.util.List<?>,java.lang.reflect.Type> value){takeNested(value);}
 static void wildcard(java.util.HashMap<Integer,String> value){takeWildcard(value);}
 static void raw(java.util.HashMap value){takeMap(value);}
 void classObject(Map value){takeObject(value);}
 <HashMap> void methodObject(HashMap value){takeObject(value);}
 static <M extends java.util.Map<String,Integer>> void bounded(M value){takeMap(value);}
 static <Caller> void callee(Caller value){takeE(value);}
 class Captured { void captured(Map value){takeObject(value);} }
 public static void main(String[] args){
  int seed=Integer.parseInt(args[0]);
  java.util.HashMap<java.util.List<?>,java.lang.reflect.Type> a=new java.util.HashMap<>();a.put(java.util.List.of(seed),String.class);nested(a);
  java.util.HashMap<Integer,String> b=new java.util.HashMap<>();b.put(seed,"value");wildcard(b);
  java.util.HashMap<String,Integer> c=new java.util.HashMap<>();c.put("seed",seed);raw(c);bounded(c);
  Use<String> use=new Use<>();use.classObject("value");use.methodObject(seed);callee(seed);use.new Captured().captured("captured");
  System.out.println("marks:"+marks+":"+seed);
 }
}
`

func TestDeclaredOriginRefusalExistingPositiveCalls(t *testing.T) {
	for _, row := range []struct {
		caller, target string
		captured       bool
	}{
		{"nested", "takeNested", false}, {"wildcard", "takeWildcard", false}, {"raw", "takeMap", false},
		{"classObject", "takeObject", false}, {"methodObject", "takeObject", false}, {"bounded", "takeMap", false},
		{"callee", "takeE", false}, {"captured", "takeObject", true},
	} {
		t.Run(row.caller, func(t *testing.T) {
			ctx := systematicNegativeFullPipelineContext(t, declaredOriginRefusalPositiveJava)
			root := ctx.currentClass
			callerOwner := root
			if row.captured {
				callerOwner = nil
				for _, scope := range root.Subclasses {
					if scope.Class.OriginalName == "Captured" {
						callerOwner = scope
					}
				}
				if callerOwner == nil {
					t.Fatal("actual registered captured owner missing")
				}
			}
			var caller, target *symbol.Definition
			for _, d := range callerOwner.Methods {
				if d.OriginalName == row.caller {
					caller = d
				}
			}
			for _, d := range root.Methods {
				if d.OriginalName == row.target {
					target = d
				}
			}
			if caller == nil || target == nil {
				t.Fatal("actual original methods missing")
			}
			ctx.currentClass = callerOwner
			ctx.localScope = caller
			ctx.className = callerOwner.Class.Name
			var call *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n == nil {
					return
				}
				if n.Type() == "method_invocation" {
					call = n
				}
				for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
					walk(c)
				}
			}
			walk(caller.DeclarationNode.ChildByFieldName("body"))
			if call == nil {
				t.Fatal("actual original call missing")
			}
			args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(args[0], ctx, ctx.currentFile.Source, &actual)
			actual.javaType = typ
			formals := methodFormalOrigins(target, root, nil, ctx)
			score, applies := scoreMethodCandidate(target, root, methodCandidateTypeParameterNames(root, target), args, ctx, ctx.currentFile.Source)
			t.Logf("actual=%s formal=%s score=%+v applies=%t caller=%p owner=%p originalTarget=%p", gsonParentDiagnosticOrigin(actual, nil), gsonParentDiagnosticOrigin(formals[0], nil), score, applies, caller, callerOwner, target)
			if !known || !applies {
				t.Error("actual JDK-valid original invocation must remain applicable")
			}
			if row.captured && actual.parameter != root.OwnTypeParameters()[0].Declaration {
				t.Error("captured argument lost exact original outer class declaration pointer")
			}
			if row.caller == "callee" && (actual.parameter == nil || actual.parameter != caller.TypeParameters[0].Declaration || formals[0].parameter != target.TypeParameters[0].Declaration || actual.parameter == formals[0].parameter) {
				t.Error("caller and callee inference binders must remain distinct original declarations")
			}
		})
	}
}
