package transpiler

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestSourceInterfaceVirtualEntryCollisionJDK(t *testing.T) {
	// Legal source selectors occupy three successive candidates before the
	// corresponding local and anonymous definitions have been hoisted.
	base := "Java2goInterfaceVirtual_" + strings.ToUpper(hex.EncodeToString([]byte("dispatch.Main$View\x00value\x01java.lang.Object")))
	source := `package dispatch;public class Main {
 interface View {Object value();}
 static class D extends R {public Object NAME; public String value(){return "child";}}
 static class R implements View {public Object value(){return "base";}}
 static String run(){class L extends R {public Object NAME_1(){return null;}public String value(){return "local";}}R l=new L();View lv=l;R a=new R(){public Object NAME_2(){return null;}public String value(){return "anon";}};View av=a;R r=new D();View v=r;return v.value()+":"+lv.value()+":"+av.value()+":"+new R().value();}
 public static void main(String[]args){System.out.println(run());}
 }`
	source = strings.ReplaceAll(source, "NAME", base)
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"src/main/java/dispatch/Main.java": source, "pom.xml": "<project><groupId>dispatch</groupId><artifactId>dispatch</artifactId><version>1</version></project>"}, "dispatch.Main", []campaignCompilerProjectObservation{{name: "first", stdout: "child:local:anon:base\n"}, {name: "second", stdout: "child:local:anon:base\n"}, {name: "third", stdout: "child:local:anon:base\n"}})
}

func TestSourceInterfaceVirtualEntryMultiPackageJDK(t *testing.T) {
	files := map[string]string{
		"pom.xml":                          "<project><groupId>dispatch</groupId><artifactId>dispatch</artifactId><version>1</version></project>",
		"src/main/java/api/View.java":      `package api;public interface View {base.Value value(base.Value __java2goInterfaceParent);}`,
		"src/main/java/base/Value.java":    `package base;public class Value {public int code;public Value(int n){code=n;}}`,
		"src/main/java/base/Parent.java":   `package base;public class Parent implements api.View {public Value value(Value input){return input;}}`,
		"src/main/java/derived/Child.java": `package derived;public class Child extends base.Parent {public base.Value value(base.Value input){return new base.Value(input.code+7);}}`,
		"src/main/java/app/Value.java":     `package app;class Value {int code=90;}`,
		"src/main/java/app/Main.java":      `package app;public class Main {interface Getter {base.Value get(base.Value x);}public static void main(String[]args){base.Parent p=new derived.Child();api.View v=p;base.Value x=new base.Value(2);Getter ref=v::value;System.out.println(v.value(x).code+":"+ref.get(x).code+":"+x.code+":"+new Value().code);}}`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{{name: "first", stdout: "9:9:2:90\n"}, {name: "second", stdout: "9:9:2:90\n"}, {name: "third", stdout: "9:9:2:90\n"}})
}

func TestSourceInterfaceVirtualEntryReferenceNullJDK(t *testing.T) {
	files := map[string]string{"pom.xml": "<project><groupId>dispatch</groupId><artifactId>dispatch</artifactId><version>1</version></project>", "src/main/java/dispatch/Main.java": `package dispatch;public class Main {
 interface View {int value(int n);} interface Bound {int value(Integer n);}interface Unbound {int value(View v,Integer n);}
 static class R implements View {public int value(int n){return n+1;}}static class D extends R {public int value(int n){return n+7;}}
 static int calls;static View receiver(boolean absent){calls++;R r=new D();return absent?null:r;}
 public static void main(String[]args){View v=receiver(false);Bound b=v::value;Unbound u=View::value;System.out.println(b.value(2)+":"+u.value(v,3)+":"+calls);try{Bound bad=receiver(true)::value;}catch(NullPointerException e){System.out.println("bound="+calls);}try{u.value(null,2);}catch(NullPointerException e){System.out.println("unbound="+calls);}try{u.value(v,null);}catch(NullPointerException e){System.out.println("argument="+calls);}try{u.value(null,null);}catch(NullPointerException e){System.out.println("both="+calls);}}
 }`}
	want := "9:10:1\nbound=2\nunbound=2\nargument=2\nboth=2\n"
	runCampaignCompilerStrictProjectObservations(t, files, "dispatch.Main", []campaignCompilerProjectObservation{{name: "first", stdout: want}, {name: "second", stdout: want}, {name: "third", stdout: want}})
}
