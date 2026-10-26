package transpiler

import "testing"

func TestMathFloorModPrimitiveOverloadsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/floormod/Main.java": `package floormod;
public class Main {
 static void ii(String label,int x,int y){try{int result=Math.floorMod(x,y);System.out.println("ii/"+label+"="+result);}catch(ArithmeticException error){System.out.println("ii/"+label+"="+error.getClass().getName()+":"+error.getMessage());}}
 static void li(String label,long x,int y){try{int result=Math.floorMod(x,y);System.out.println("li/"+label+"="+result);}catch(ArithmeticException error){System.out.println("li/"+label+"="+error.getClass().getName()+":"+error.getMessage());}}
 static void ll(String label,long x,long y){try{long result=Math.floorMod(x,y);System.out.println("ll/"+label+"="+result);}catch(ArithmeticException error){System.out.println("ll/"+label+"="+error.getClass().getName()+":"+error.getMessage());}}
 static String pick(int value){return "int:"+value;}
 static String pick(long value){return "long:"+value;}
 public static void main(String[] args){
    ii("pp",16,3);
    ii("np",-16,3);
    ii("pn",16,-3);
    ii("nn",-16,-3);
    ii("zero",0,3);
    ii("min-minus-one",Integer.MIN_VALUE,-1);
    ii("min-positive",Integer.MIN_VALUE,3);
    ii("max-min",Integer.MAX_VALUE,Integer.MIN_VALUE);
    ii("min-min",Integer.MIN_VALUE,Integer.MIN_VALUE);
    ii("zero-divisor",1,0);
    ii("zero-zero",0,0);
    li("wide",9007199254740993L,7);
    li("min-positive",Long.MIN_VALUE,3);
    li("max-min-int",Long.MAX_VALUE,Integer.MIN_VALUE);
    li("min-minus-one",Long.MIN_VALUE,-1);
    li("negative-min-int",-1L,Integer.MIN_VALUE);
    li("positive-min-int",1L,Integer.MIN_VALUE);
    li("max-max-int",Long.MAX_VALUE,Integer.MAX_VALUE);
    li("zero-divisor",Long.MIN_VALUE,0);
    ll("pp",16L,3L);
    ll("np",-16L,3L);
    ll("pn",16L,-3L);
    ll("nn",-16L,-3L);
    ll("precision",9007199254740993L,7L);
    ll("min-minus-one",Long.MIN_VALUE,-1L);
    ll("max-min",Long.MAX_VALUE,Long.MIN_VALUE);
    ll("one-min",1L,Long.MIN_VALUE);
    ll("min-min",Long.MIN_VALUE,Long.MIN_VALUE);
    ll("zero-divisor",Long.MIN_VALUE,0L);
  System.out.println("result-li="+pick(Math.floorMod(Long.MIN_VALUE,3)));
  System.out.println("result-il="+pick(Math.floorMod(1,Long.MIN_VALUE)));
 }
}
`,
	}, "floormod.Main", "ii/pp=1\nii/np=2\nii/pn=-2\nii/nn=-1\nii/zero=0\nii/min-minus-one=0\nii/min-positive=1\nii/max-min=-1\nii/min-min=0\nii/zero-divisor=java.lang.ArithmeticException:/ by zero\nii/zero-zero=java.lang.ArithmeticException:/ by zero\nli/wide=5\nli/min-positive=1\nli/max-min-int=-1\nli/min-minus-one=0\nli/negative-min-int=-1\nli/positive-min-int=-2147483647\nli/max-max-int=1\nli/zero-divisor=java.lang.ArithmeticException:/ by zero\nll/pp=1\nll/np=2\nll/pn=-2\nll/nn=-1\nll/precision=5\nll/min-minus-one=0\nll/max-min=-1\nll/one-min=-9223372036854775807\nll/min-min=0\nll/zero-divisor=java.lang.ArithmeticException:/ by zero\nresult-li=int:1\nresult-il=long:-9223372036854775807\n")
}

func TestMathFloorModConversionsAndOrderJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/floormod/Main.java": `package floormod;
import static java.lang.Math.floorMod;
public class Main {
 static String trace="";
 static int i(String label,int value){trace+=label;return value;}
 static long l(String label,long value){trace+=label;return value;}
 static Integer absent(){trace+="n";return null;}
 static String pick(int value){return "int:"+value;}
 static String pick(long value){return "long:"+value;}
 public static void main(String[] args){
  byte b=-7;short s=3;char c='A';Integer left=7;Integer right=-3;Long wide=-7L;Long divisor=-3L;
  System.out.println("promotion="+floorMod(b,s)+":"+floorMod(c,s)+":"+floorMod(left,right)+":"+floorMod(wide,s)+":"+floorMod(left,divisor));
  System.out.println("selected="+pick(floorMod(wide,s))+":"+pick(floorMod(left,divisor)));
  trace="";int x=floorMod(i("a",-7),i("b",3));System.out.println("order-int="+x+":"+trace);
  trace="";int y=floorMod(l("c",Long.MIN_VALUE),i("d",3));System.out.println("order-mixed="+y+":"+trace);
  trace="";long z=floorMod(i("e",7),l("f",-3L));System.out.println("order-long="+z+":"+trace);
  trace="";try{floorMod(i("g",7),i("h",0));}catch(ArithmeticException error){System.out.println("zero="+trace+":"+error.getMessage());}
  trace="";try{floorMod(absent(),i("r",3));}catch(NullPointerException error){System.out.println("first-null="+trace);}
  trace="";try{floorMod(i("p",3),absent());}catch(NullPointerException error){System.out.println("second-null="+trace);}
 }
}
`,
	}, "floormod.Main", "promotion=2:2:-2:2:-2\nselected=int:2:long:-2\norder-int=2:ab\norder-mixed=1:cd\norder-long=-2:ef\nzero=gh:/ by zero\nfirst-null=n\nsecond-null=pn\n")
}

func TestMathFloorModMethodReferencesJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/floormod/Main.java": `package floormod;
import java.util.function.BiFunction;
public class Main {
 interface II {int run(int x,int y);}
 interface LI {int run(long x,int y);}
 interface LL {long run(long x,long y);}
 interface IL {long run(int x,long y);}
 interface Widen {long run(long x,int y);}
 interface NarrowArgs {int run(byte x,short y);}
 public static void main(String[] args){
  II ii=Math::floorMod;LI li=java.lang.Math::floorMod;LL ll=Math::floorMod;IL il=Math::floorMod;Widen widened=Math::floorMod;NarrowArgs narrow=Math::floorMod;
  System.out.println("primitive="+ii.run(-7,3)+":"+li.run(Long.MIN_VALUE,3)+":"+ll.run(1L,Long.MIN_VALUE)+":"+il.run(7,-3L)+":"+widened.run(Long.MIN_VALUE,3)+":"+narrow.run((byte)-7,(short)3));
  BiFunction<Integer,Integer,Integer> bii=Math::floorMod;
  BiFunction<Long,Integer,Integer> bli=Math::floorMod;
  BiFunction<Long,Long,Long> bll=Math::floorMod;
  BiFunction<Integer,Long,Long> bil=Math::floorMod;
  System.out.println("boxed="+bii.apply(-7,3)+":"+bli.apply(Long.MIN_VALUE,3)+":"+bll.apply(1L,Long.MIN_VALUE)+":"+bil.apply(7,-3L));
  try{ii.run(1,0);}catch(ArithmeticException error){System.out.println("zero="+error.getClass().getName()+":"+error.getMessage());}
  Integer absent=null;try{bii.apply(absent,3);}catch(NullPointerException error){System.out.println("null=java.lang.NullPointerException");}
 }
}
`,
	}, "floormod.Main", "primitive=2:1:-9223372036854775807:-2:1:2\nboxed=2:1:-9223372036854775807:-2\nzero=java.lang.ArithmeticException:/ by zero\nnull=java.lang.NullPointerException\n")
}

func TestMathFloorModSourceAndValueShadowsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/floormod/Main.java": `package floormod;
import static java.lang.Math.*;
class Math {static int floorMod(int x,int y){return 17;}}
class Fake {int calls;int floorMod(int x,int y){calls++;return 23;}}
class ValueScope {Fake Math=new Fake();int run(){int result=Math.floorMod(-7,3);return result+Math.calls;}}
public class Main {
 public static void main(String[] args){System.out.println("source="+Math.floorMod(-7,3)+":"+java.lang.Math.floorMod(-7,3)+":"+floorMod(-7,3));System.out.println("value="+new ValueScope().run());}
}
`,
	}, "floormod.Main", "source=17:2:2\nvalue=24\n")
}
