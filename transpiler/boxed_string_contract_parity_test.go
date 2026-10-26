package transpiler

import "testing"

func boxedStringStrictParity(t *testing.T, class, source string, extra map[string]string) {
	t.Helper()
	want := stringCaseReferenceJavaOracle(t, class, source)
	files := map[string]string{"pom.xml": `<project><groupId>example</groupId><artifactId>boxed-string</artifactId><version>1</version></project>`, "src/main/java/" + class + ".java": source}
	for path, text := range extra {
		files[path] = text
	}
	runCampaignCompilerStrictProjectOracle(t, files, class, want)
}

func TestBoxedStringFactoriesConstructorsStrictJDK21Parity(t *testing.T) {
	const source = `public class BoxedStringFactories {
 public static String run(){
 Boolean bool=Boolean.valueOf("TrUe");Byte small=Byte.valueOf("７");Short narrow=Short.valueOf("７");Integer integer=Integer.valueOf("７");Long longer=Long.valueOf("７");Float single=Float.valueOf("0x1.8p1f");Double decimal=Double.valueOf("0x1.8p1D");
 return (bool==Boolean.TRUE)+":"+(small==Byte.valueOf((byte)7))+":"+(narrow==Short.valueOf((short)7))+":"+(integer==Integer.valueOf(7))+":"+(longer==Long.valueOf(7))+":"+single+":"+decimal
 +":"+(new Boolean("true")!=bool)+":"+(new Byte("７")!=small)+":"+(new Short("７")!=narrow)+":"+(new Integer("７")!=integer)+":"+(new Long("７")!=longer)+":"+(new Float("3.0")!=single)+":"+(new Double("3.0")!=decimal)
 +":"+(Boolean.valueOf((String)null)==Boolean.FALSE)+":"+new Boolean((String)null).booleanValue()+":"+Integer.valueOf("Ｆ",16)+":"+Long.valueOf("-７");
 }
 public static void main(String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "BoxedStringFactories", source, nil)
}

func TestBoxedStringReferencesStrictJDK21Parity(t *testing.T) {
	const source = `interface BooleanText {Boolean make(String value);}interface ByteText {Byte make(String value);}interface ShortText {Short make(String value);}interface IntegerText {Integer make(String value);}interface LongText {Long make(String value);}interface FloatText {Float make(String value);}interface DoubleText {Double make(String value);}interface RadixText {Integer make(String value,int radix);}
public class BoxedStringReferences {
 public static String run(){
 BooleanText b=Boolean::valueOf,bn=Boolean::new;ByteText y=Byte::valueOf,yn=Byte::new;ShortText s=Short::valueOf,sn=Short::new;IntegerText i=Integer::valueOf,in=Integer::new;LongText l=Long::valueOf,ln=Long::new;FloatText f=Float::valueOf,fn=Float::new;DoubleText d=Double::valueOf,dn=Double::new;RadixText r=Integer::valueOf;
 return (b.make("true")==Boolean.TRUE)+":"+(y.make("７")==Byte.valueOf((byte)7))+":"+(s.make("７")==Short.valueOf((short)7))+":"+(i.make("７")==Integer.valueOf(7))+":"+(l.make("７")==Long.valueOf(7))+":"+f.make("1.5")+":"+d.make("2.5")
 +":"+(bn.make("true")!=Boolean.TRUE)+":"+(yn.make("７")!=y.make("７"))+":"+(sn.make("７")!=s.make("７"))+":"+(in.make("７")!=i.make("７"))+":"+(ln.make("７")!=l.make("７"))+":"+(fn.make("1.5")!=f.make("1.5"))+":"+(dn.make("2.5")!=d.make("2.5"))+":"+r.make("Ｆ",16)+":"+(b.make(null)==Boolean.FALSE)+":"+bn.make(null).booleanValue();
 }
 public static void main(String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "BoxedStringReferences", source, nil)
}

func TestBoxedStringForeignOwnersStrictJDK21Parity(t *testing.T) {
	files := map[string]string{
		"pom.xml":                               `<project><groupId>example</groupId><artifactId>boxed-string-foreign</artifactId><version>1</version></project>`,
		"src/main/java/foreign/String.java":     `package foreign;public class String {public String(){}public java.lang.String toString(){return "foreign-text";}}`,
		"src/main/java/foreign/Byte.java":       `package foreign;public class Byte {java.lang.String marker;public Byte(String value){marker="foreign:"+value;}public static Byte valueOf(String value){return new Byte(value);}public java.lang.String toString(){return marker;}}`,
		"src/main/java/BoxedStringForeign.java": `import foreign.Byte;import foreign.String;interface ForeignFactory {Byte make(String value);}public class BoxedStringForeign {public static java.lang.String run(){ForeignFactory factory=Byte::valueOf,constructor=Byte::new;String value=new String();return factory.make(value)+":"+constructor.make(value)+":"+java.lang.Byte.valueOf("７")+":"+new java.lang.Byte("７");}public static void main(java.lang.String[] args){System.out.print(run());}}`,
	}
	// The strict project harness verifies this snapshot against the live JDK
	// before translating the same foreign implementations and comparing Go.
	runCampaignCompilerStrictProjectOracle(t, files, "BoxedStringForeign", "foreign:foreign-text:foreign:foreign-text:7:7")
}

func TestBoxedStringParserFailuresExecutionStrictJDK21Parity(t *testing.T) {
	const source = `public class BoxedStringParserFailures {
 static String trace="";static Object lock=new Object();static Thread owner;static RuntimeException failure=new IllegalStateException("exact");
 static String units(String value){if(value==null)return "null";String out="";for(int i=0;i<value.length();i++)out+=(int)value.charAt(i)+",";return out;}
 static String text(int mode){trace+="T";if(Thread.currentThread()!=owner||!Thread.holdsLock(lock))throw new AssertionError("text execution");return mode==0?null:mode==1?new String(new char[]{'1',(char)0xd800}):"１２";}
 static int radix(int mode){trace+="R";if(Thread.currentThread()!=owner||!Thread.holdsLock(lock))throw new AssertionError("radix execution");if(mode==2)throw failure;return mode==0||mode==3?1:10;}
 static String parse(int mode){try{if(mode==0)return ""+new Byte("１２８").byteValue();if(mode==1)return ""+new Short("３２７６８").shortValue();if(mode==2)return ""+new Integer((String)null).intValue();if(mode==3)return ""+new Long(new String(new char[]{'1',(char)0xdfff})).longValue();if(mode==4)return ""+Float.valueOf((String)null).floatValue();return ""+new Double((String)null).doubleValue();}catch(RuntimeException ex){return ex.getClass().getSimpleName()+":"+units(ex.getMessage());}}
 public static String run(){synchronized(lock){owner=Thread.currentThread();String out="";for(int mode=0;mode<4;mode++){trace="";try{Integer.valueOf(text(mode),radix(mode));out+="missing";}catch(RuntimeException ex){out+=trace+":"+(ex==failure)+":"+ex.getClass().getSimpleName()+":"+units(ex.getMessage());}out+="|";}for(int mode=0;mode<6;mode++)out+=parse(mode)+"|";return out;}}
 public static void main(String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "BoxedStringParserFailures", source, nil)
}

func TestBoxedStringSupportedSourceControlStrictJDK21Parity(t *testing.T) {
	const source = `class String {String(){} }
class Integer {java.lang.String value;Integer(java.lang.String value){this.value=value;}static Integer valueOf(java.lang.String value){return new Integer(value);}java.lang.String marker(){return "source:"+value;}}
public class BoxedStringSupportedSource {
 static <Integer> java.lang.Integer binder(Integer ignored){return new java.lang.Integer("７");}
 public static java.lang.String run(){Integer source=Integer.valueOf("owner");String sourceText=new String();return source.marker()+":"+java.lang.Integer.valueOf("７")+":"+binder(sourceText);}
 public static void main(java.lang.String[] args){System.out.print(run());}
}`
	boxedStringStrictParity(t, "BoxedStringSupportedSource", source, nil)
}
