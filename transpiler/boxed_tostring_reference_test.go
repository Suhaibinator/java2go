package transpiler

import "testing"

func TestBoxedToStringCanonicalResultsTDD(t *testing.T) {
	for _, name := range []string{"Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double", "Character"} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, "public class TextBoundary {static java.lang.String instance(java.lang."+name+" value){return value.toString();}}")
			assertNotContains(t, out, "return value.String()")
			assertContains(t, out, "stdjava.JavaStringValueOfExecution(")
		})
	}
}

func TestBoxedToStringPrimitiveAndInstanceJDK21Parity(t *testing.T) {
	const source = `public class BoxedText {
 public static String run(){
 String b=Boolean.toString(true),y=Byte.toString((byte)-7),s=Short.toString((short)-8),i=Integer.toString(-9),l=Long.toString(-10L),f=Float.toString(1.5f),d=Double.toString(2.5),c=Character.toString((char)0xd800);
 String instances=Boolean.TRUE.toString()+":"+Byte.valueOf((byte)-7).toString()+":"+Short.valueOf((short)-8).toString()+":"+Integer.valueOf(-9).toString()+":"+Long.valueOf(-10L).toString()+":"+Float.valueOf(1.5f).toString()+":"+Double.valueOf(2.5).toString();
 return b+":"+y+":"+s+":"+i+":"+l+":"+f+":"+d+"|"+instances+"|"+c.length()+":"+(int)c.charAt(0)+":"+(int)Character.valueOf((char)0xdfff).toString().charAt(0)
 +":"+(int)Character.toString((char)0xdc00).charAt(0)+":"+(Character.toString((char)0xd800)+Character.toString((char)0xdc00)).length()+"|"+(b=="true")+":"+(Boolean.TRUE.toString()=="true")+":"+(Integer.toString(7)!=Integer.toString(7))+":"+(Integer.valueOf(7).toString()!=Integer.valueOf(7).toString())+":"+(Float.toString(0.0f)=="0.0")+":"+(Double.valueOf(-0.0).toString()=="-0.0")+":"+(Float.toString(1.5f)!=Float.toString(1.5f));
 }
 public static void main(String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "BoxedText", source, nil)
}

func TestBoxedToStringEvaluationExecutionValidReferencesJDK21Parity(t *testing.T) {
	const source = `interface DoubleText {String text(double value);}interface TextSupplier {String get();}
public class BoxedTextEffects {
 static Object lock=new Object();static Thread owner;static String trace="";static RuntimeException failure=new IllegalStateException("same");
 static void check(){if(Thread.currentThread()!=owner||!Thread.holdsLock(lock))throw new AssertionError("execution");}
 static double primitive(int mode){check();trace+="P";if(mode==1)throw failure;return 1.5;}
 static Double boxed(int mode){check();trace+="B";if(mode==1)throw failure;return mode==2?null:Double.valueOf(2.5);}
 public static String run(){synchronized(lock){owner=Thread.currentThread();String out=Double.toString(primitive(0))+":"+boxed(0).toString()+":"+trace;trace="";
 DoubleText staticRef=Double::toString;TextSupplier bound=boxed(0)::toString;
 out+="|"+staticRef.text(primitive(0))+":"+bound.get()+":"+trace;trace="";
 try{Double.toString(primitive(1));out+="missing";}catch(RuntimeException ex){out+="|"+(ex==failure)+":"+trace;}trace="";
 try{boxed(1).toString();out+="missing";}catch(RuntimeException ex){out+="|"+(ex==failure)+":"+trace;}trace="";
 try{boxed(2).toString();out+="missing";}catch(NullPointerException ex){out+="|null-instance:"+trace;}trace="";
 try{TextSupplier rejected=boxed(2)::toString;out+="missing";}catch(NullPointerException ex){out+="|null-bound:"+trace;}
 try{Double.toString((Double)null);out+="missing";}catch(NullPointerException ex){out+="|null-static";}
 return out;}}
 public static void main(String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "BoxedTextEffects", source, nil)
}

func TestBoxedToStringSourceAndForeignOwnersJDK21Parity(t *testing.T) {
	const source = `class Double {public String toString(){return "source";}static String toString(int value){return "local:"+value;}}
class TextNamespace {TextLanguage lang=new TextLanguage();}class TextLanguage {TextLeaf Double=new TextLeaf();}class TextLeaf {static String toString(double value){return "valuepath:"+value;}}
public class BoxedTextSource {static <Double> String binder(Double value){return value.toString();}static String namespace(TextNamespace java){return java.lang.Double.toString(2.5);}public static String run(){Double source=new Double();return source.toString()+":"+Double.toString(7)+":"+binder(source)+":"+java.lang.Double.toString(2.5)+":"+namespace(new TextNamespace());}public static void main(String[] args){System.out.print(run());}}`
	boxedStringStrictParity(t, "BoxedTextSource", source, nil)
	files := map[string]string{
		"pom.xml":                             `<project><groupId>example</groupId><artifactId>foreign-boxed-text</artifactId><version>1</version></project>`,
		"src/main/java/foreign/Double.java":   `package foreign;public class Double {public java.lang.String toString(){return "foreign";}public static java.lang.String toString(int value){return "foreign:"+value;}}`,
		"src/main/java/ForeignBoxedText.java": `import foreign.Double;public class ForeignBoxedText {public static String run(){Double value=new Double();return value.toString()+":"+Double.toString(7)+":"+java.lang.Double.toString(2.5);}public static void main(String[] args){System.out.print(run());}}`,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "ForeignBoxedText", "foreign:foreign:7:2.5")
}

// These declaration-only controls test refusal; the binder source is intentionally
// not a JVM-valid invocation and is never presented as a Java runtime oracle.
func TestBoxedToStringOwnerGuardsTDD(t *testing.T) {
	for _, name := range []string{"Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double", "Character"} {
		t.Run(name, func(t *testing.T) {
			cases := []struct{ name, source string }{
				{"foreign", "import foreign." + name + ";public class TextOwner {Object read(){return " + name + ".toString(1);}}"},
				{"source", "class " + name + " {static java.lang.String toString(int x){return \"own\";}}public class TextOwner {Object read(){return " + name + ".toString(1);}}"},
				{"binder", "public class TextOwner<" + name + "> {Object read(){return " + name + ".toString(1);}}"},
				{"package-binder", "public class TextOwner<java> {Object read(){return java.lang." + name + ".toString(1);}}"},
				{"package-value", "public class TextOwner {Object java;Object read(){return java.lang." + name + ".toString(1);}}"},
				{"relative-package-binder", "public class TextOwner<lang> {Object read(){return lang." + name + ".toString(1);}}"},
				{"relative-package-value", "public class TextOwner {Object lang;Object read(){return lang." + name + ".toString(1);}}"},
			}
			for _, control := range cases {
				t.Run(control.name, func(t *testing.T) {
					strictRoutingState(t)
					helper := setupParseHelper(t, control.source)
					invocation := findNode(helper.File.Ast, "method_invocation")
					expression, mapped := tryStaticIntrinsic(invocation.ChildByFieldName("object"), "toString", helper.File.Source, helper.Ctx)
					if mapped || expression != nil || len(Diagnostics()) != 0 {
						t.Fatalf("non-JDK declaration borrowed boxed text intrinsic: expression=%v mapped=%t diagnostics=%v", expression, mapped, Diagnostics())
					}
				})
			}
		})
	}
}

func TestBoxedToStringReferencesAllWrappersJDK21Parity(t *testing.T) {
	const source = `interface BoolText {String call(boolean x);}interface ByteText {String call(byte x);}interface ShortText {String call(short x);}interface IntText {String call(int x);}interface LongText {String call(long x);}interface FloatText {String call(float x);}interface DoubleText {String call(double x);}interface CharText {String call(char x);}interface TextSupplier {String call();}
public class AllBoxedTextReferences {
 public static String run(){
 BoolText b=Boolean::toString;ByteText y=Byte::toString;ShortText s=Short::toString;IntText i=Integer::toString;LongText l=Long::toString;FloatText f=Float::toString;DoubleText d=Double::toString;CharText c=Character::toString;
 TextSupplier bn=Boolean.TRUE::toString,yn=Byte.valueOf((byte)7)::toString,sn=Short.valueOf((short)8)::toString,in=Integer.valueOf(9)::toString,ln=Long.valueOf(10L)::toString,fn=Float.valueOf(1.5f)::toString,dn=Double.valueOf(2.5)::toString,cn=Character.valueOf((char)0xdc00)::toString;
 String chars=c.call((char)0xd800)+cn.call();
 return b.call(true)+":"+y.call((byte)7)+":"+s.call((short)8)+":"+i.call(9)+":"+l.call(10L)+":"+f.call(1.5f)+":"+d.call(2.5)+"|"+bn.call()+":"+yn.call()+":"+sn.call()+":"+in.call()+":"+ln.call()+":"+fn.call()+":"+dn.call()+"|"+chars.length()+":"+(int)chars.charAt(0)+":"+(int)chars.charAt(1);
 }
 public static void main(String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "AllBoxedTextReferences", source, nil)
}
