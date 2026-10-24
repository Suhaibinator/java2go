package transpiler

import "testing"

func TestSourceInterfaceVirtualEntryJDK(t *testing.T) {
	for _, test := range []struct{ name, source, want string }{
		{"methodReferences", `package dispatch;public class Main {
 interface View {Object value();} interface Getter {Object get();} interface From {Object get(View v);}
 static class R implements View {public Object value(){return "parent";}}
 static class D extends R {public String value(){return "child";}}
 static View receive(){return new D();}
 public static void main(String[]args){R r=new D();View v=r;Getter bound=v::value;From unbound=View::value;System.out.println(bound.get()+":"+unbound.get(v)+":"+receive().value());}
 }`, "child:child:child\n"},
		{"inheritedOwnerAndDefault", `package dispatch;public class Main {
 interface Root {Object value();default Object fallback(){return "default";}}
 interface View extends Root {Object value(int n);void put(int a,int b);}
 static class B {public Object value(){return "base";}}
 static class R extends B implements View {public Object value(int n){return "over";} public void put(int a,int b){} }
 static class D extends R {public String value(){return "child";} String parent(){return (String)super.value();}}
 public static void main(String[]args){R r=new D();View v=r;System.out.println(v.value()+":"+v.value(3)+":"+v.fallback()+":"+new D().parent());}
 }`, "child:over:default:base\n"},
		{"orderNullVoidAndExecution", `package dispatch;public class Main {
 interface View {Object value(int a,int b);void put(int a,int b);Object empty();boolean held();}
 static int trace,bodies;static boolean absent;static Object monitor=new Object();
 static class R implements View {public Object value(int a,int b){bodies++;return "base";}public void put(int a,int b){bodies++;}public Object empty(){return null;}public boolean held(){return Thread.holdsLock(monitor);}}
 static class D extends R {public String value(int a,int b){bodies++;return "child";}public void put(int a,int b){bodies+=2;}}
 static View receiver(){trace=trace*10+1;R r=new D();return absent?null:r;}
 static int argument(int n){trace=trace*10+n;return n;}
 public static void main(String[]args){System.out.println(receiver().value(argument(2),argument(3))+":"+trace+":"+bodies);trace=0;receiver().put(argument(2),argument(3));System.out.println(trace+":"+bodies);trace=0;absent=true;try{receiver().value(argument(2),argument(3));}catch(NullPointerException e){System.out.println("null="+trace+":"+bodies);}finally{System.out.println("finally="+trace);}absent=false;View v=receiver();synchronized(monitor){System.out.println((v.empty()==null)+":"+v.held());}}
 }`, "child:123:1\n123:3\nnull=123:3\nfinally=123\ntrue:true\n"},
		{"localParentAndAnonymous", `package dispatch;public class Main {
 interface View {Object value();}
 static String run(){class R implements View {public Object value(){return "localbase";}}class D extends R {public String value(){return "localchild";}}R r=new D();View v=r;R a=new R(){public String value(){return "anon";}};View av=a;return v.value()+":"+av.value()+":"+new R().value();}
 public static void main(String[]args){System.out.println(run());}
 }`, "localchild:anon:localbase\n"},
		{"arraysVarargsAndNoOverride", `package dispatch;public class Main {
 interface View {Object[] values(Object[] a);String text(String... a);}
 static class R implements View {public Object[] values(Object[] a){return a;}public String text(String... a){return "base"+a.length;}}
 static class D extends R {public String text(String... a){return "child"+a.length;}}
 public static void main(String[]args){R r=new D();View v=r;Object[] a={"first",null};System.out.println((v.values(a)==a)+":"+(v.values(a)[1]==null)+":"+v.text("a","b")+":"+new R().text("a"));}
 }`, "true:true:child2:base1\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runCampaignCompilerStrictProjectObservations(t, map[string]string{"src/main/java/dispatch/Main.java": test.source, "pom.xml": "<project><groupId>dispatch</groupId><artifactId>dispatch</artifactId><version>1</version></project>"}, "dispatch.Main", []campaignCompilerProjectObservation{{name: "first", stdout: test.want}, {name: "second", stdout: test.want}, {name: "third", stdout: test.want}})
		})
	}
}
