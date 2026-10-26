package transpiler

import "testing"

func TestGenericInterfaceABILateCastBoxingJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-expanded</artifactId><version>1</version></project>`, "src/main/java/edge/Main.java": `package edge;
class java{static class lang{static class Object{int code=9;}}}
class Object{}
class Child extends java.lang.Object{}
class Alien{}
interface Poly{<T>T id(T value);}
class Named implements Poly{public<T>T id(T value){Main.calls++;return value;}}
class Polluted implements Poly{public<T>T id(T value){Main.reads++;T result=(T)new Alien();Main.bodies++;return result;}}
public class Main{static int receivers,numbers,calls,reads,bodies;static Poly receiver(Poly p){receivers++;return p;}static int number(){numbers++;return 7;}
 public static void main(String[]args){Poly p=new Named();Child child=new Child();Child same=p.id(child);System.out.println((same==child)+":"+same.code);Integer boxed=receiver(p).id(number());System.out.println(boxed+":"+receivers+":"+numbers+":"+calls);
 try{Child wrong=new Polluted().id(child);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("cast="+reads+":"+bodies);}finally{System.out.println("finally="+bodies);}
 try{Integer wrong=receiver(null).id(number());System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null="+receivers+":"+numbers+":"+calls);}
 }
}
`}, "edge.Main", "true:9\n7:1:1:2\ncast=1:1\nfinally=1\nnull=2:2:2\n")
}
func TestGenericInterfaceABIAnonymousObjectJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-expanded</artifactId><version>1</version></project>`, "src/main/java/anon/Main.java": `package anon;class Object{}class Value{}interface Poly{<T>T id(T value);}public class Main{static int calls;public static void main(String[]args){Poly p=new Poly(){public<T>T id(T value){Main.calls++;return value;}};String text=p.id("anon");Value value=new Value();Value same=p.id(value);Value nil=p.id((Value)null);Integer boxed=p.id(8);System.out.println(text+":"+(same==value)+":"+(nil==null)+":"+boxed+":"+calls);}}
`}, "anon.Main", "anon:true:true:8:4\n")
}
func TestGenericInterfaceABITopLevelFirstNestedJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-expanded</artifactId><version>1</version></project>`, "src/main/java/controls/Main.java": `package controls;
class Outer{static class Node{int value;Node(int value){this.value=value;}int code(){return value+100;}}}
class Node{int value;Node(int value){this.value=value;}int code(){return value+10;}}
enum Flag{ONE;int code(){return 1;}}record Stamp(int value){}interface Label{default int code(){return 2;}}class Named implements Label{}public class Main{public static void main(String[]args){controls.Node first=new controls.Node(3);controls.Outer.Node second=new controls.Outer.Node(4);System.out.println(first.code()+":"+second.code()+":"+first.getClass().getName()+":"+second.getClass().getName()+":"+Flag.ONE.code()+":"+new Stamp(5).value()+":"+new Named().code());}}
`}, "controls.Main", "13:104:controls.Node:controls.Outer$Node:1:5:2\n")
}
func TestGenericInterfaceABITopLevelReversedJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-expanded</artifactId><version>1</version></project>`, "src/main/java/controls/Main.java": `package controls;
class Node{int value;Node(int value){this.value=value;}int code(){return value+10;}}
class Outer{static class Node{int value;Node(int value){this.value=value;}int code(){return value+100;}}}
enum Flag{ONE;int code(){return 1;}}record Stamp(int value){}interface Label{default int code(){return 2;}}class Named implements Label{}public class Main{public static void main(String[]args){controls.Node first=new controls.Node(3);controls.Outer.Node second=new controls.Outer.Node(4);System.out.println(first.code()+":"+second.code()+":"+first.getClass().getName()+":"+second.getClass().getName()+":"+Flag.ONE.code()+":"+new Stamp(5).value()+":"+new Named().code());}}
`}, "controls.Main", "13:104:controls.Node:controls.Outer$Node:1:5:2\n")
}
