package transpiler

import "testing"

func TestGenericInterfaceABIAnonymousBodyTimingJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-body</artifactId><version>1</version></project>`, "src/main/java/body/Main.java": `package body;class java{static class lang{static class Object{int code=9;}}}class Object{}class Child extends java.lang.Object{}class Alien{}interface Poly{<T>T id(T value);}
public class Main{static int receivers,numbers,calls,reads,bodies;static Poly receiver(Poly p){receivers++;return p;}static int number(){numbers++;return 7;}static<Object>Poly identity(){return new Poly(){public<Object>Object id(Object value){calls++;return value;}};}static Poly corrupted(){return new Poly(){public<U>U id(U value){reads++;U result=(U)new Alien();bodies++;return result;}};}
public static void main(String[]args){Poly p=identity();Poly corrupt=corrupted();Child child=new Child();Child same=p.id(child);Child nil=p.id((Child)null);Integer boxed=receiver(p).id(number());System.out.println((same==child)+":"+same.code+":"+(nil==null)+":"+boxed+":"+receivers+":"+numbers+":"+calls);
try{Child wrong=corrupt.id(child);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("cast="+reads+":"+bodies);}finally{System.out.println("finally="+bodies);}
try{Integer wrong=corrupt.id(number());System.out.println("wrong");}catch(ClassCastException expected){System.out.println("boxcast="+reads+":"+bodies+":"+numbers);}
try{Integer wrong=receiver(null).id(number());System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null="+receivers+":"+numbers+":"+calls);}}
}
`}, "body.Main", "true:9:true:7:1:1:3\ncast=1:1\nfinally=1\nboxcast=2:2:2\nnull=2:3:3\n")
}
