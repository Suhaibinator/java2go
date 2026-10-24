package transpiler

import "testing"

// Proposed migration helpers only: Java validates each observation before the
// direct runtime driver is built. No generated-ABI acceptance follows from this.
func TestCampaignStringBoundaryRequireAndSwitchKey(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringBoundaryKeys", `import java.util.Objects;
public class StringBoundaryKeys {
 static String key(String s) {Objects.requireNonNull(s);StringBuilder out=new StringBuilder();for(int i=0;i<s.length();i++){int u=s.charAt(i);out.append((u>>>8)&255).append(',').append(u&255).append(';');}return out.toString();}
 static int select(String s) {return switch(s) {case "A" -> 1;case "\uD800" -> 2;case "\uFFFD" -> 3;default -> 4;};}
 public static void main(String[] args) {
  String value=new String("A");boolean required=Objects.requireNonNull(value)==value;
  boolean requireNull=false,switchNull=false;
  try {Objects.requireNonNull((String)null);}catch(NullPointerException expected){requireNull=true;}
  try {select(null);}catch(NullPointerException expected){switchNull=true;}
  String[] values={"", "A",new String(new char[]{0,(char)0xD800,(char)0xDC00,(char)0xFFFD}),new String(new char[]{(char)0xD83D,(char)0xDE03})};
  StringBuilder out=new StringBuilder();for(String s:values)out.append('[').append(key(s)).append(']');
  System.out.print(required+":"+requireNull+":"+switchNull+":"+select(value)+":"+select(new String(new char[]{(char)0xD800}))+":"+select("\uFFFD")+":"+out);
 }
}`, `package main
import("fmt";"strings";j "github.com/NickyBoy89/java2go/stdjava")
func selectKey(s *j.JavaString)int{switch j.JavaStringSwitchKey(s){case "\x00A":return 1;case "\xD8\x00":return 2;case "\xFF\xFD":return 3;default:return 4}}
func main(){
 value:=j.NewJavaStringUTF16([]uint16{'A'})
 required:=j.RequireJavaString(value)==value
 npe:=func(f func())(ok bool){defer func(){ok=j.CaughtAs(recover(),"NullPointerException")}();f();return}
 requireNull:=npe(func(){j.RequireJavaString(nil)});switchNull:=npe(func(){selectKey(nil)})
 values:=[]*j.JavaString{j.NewJavaStringUTF16(nil),value,j.NewJavaStringUTF16([]uint16{0,0xD800,0xDC00,0xFFFD}),j.NewJavaStringUTF16([]uint16{0xD83D,0xDE03})}
 var out strings.Builder
 for _,s:=range values{out.WriteByte('[');key:=j.JavaStringSwitchKey(s);for i:=0;i<len(key);i+=2{fmt.Fprintf(&out,"%d,%d;",key[i],key[i+1])};out.WriteByte(']')}
 fmt.Printf("%t:%t:%t:%d:%d:%d:%s",required,requireNull,switchNull,selectKey(value),selectKey(j.NewJavaStringUTF16([]uint16{0xD800})),selectKey(j.NewJavaStringUTF16([]uint16{0xFFFD})),out.String())
}`)
}

func TestCampaignStringBoundaryTextOperandExecution(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringBoundaryOperand", `public class StringBoundaryOperand {
 static final class Stored {final String value;final Thread caller;final boolean fail;int calls;boolean same;
  Stored(String value,Thread caller,boolean fail){this.value=value;this.caller=caller;this.fail=fail;}
  public String toString(){calls++;same=Thread.currentThread()==caller;if(fail)throw new IllegalStateException();return value;}}
 public static void main(String[] args)throws Exception{
  String value=new String(new char[]{'Q',(char)0xD800});Thread creator=Thread.currentThread();StringBuilder result=new StringBuilder();
  Thread thread=new Thread(()->{
   Stored source=new Stored(value,Thread.currentThread(),false),absent=new Stored(null,Thread.currentThread(),false),failure=new Stored(value,Thread.currentThread(),true);
   String direct=String.valueOf((Object)absent);String nullText=""+absent;String storedText=""+source;boolean identity=String.valueOf((Object)source)==value;
   boolean caught=false;try{String ignored=""+failure;}catch(IllegalStateException expected){caught=true;}
   String ordinaryNull=""+(Object)null;
   result.append(direct==null).append(':').append(nullText.equals("null")).append(':').append(storedText.equals(value)&&storedText!=value)
    .append(':').append(identity).append(':').append(ordinaryNull.equals("null")).append(':').append(caught)
    .append(':').append(source.same&&absent.same&&failure.same&&Thread.currentThread()!=creator)
    .append(':').append(source.calls).append(':').append(absent.calls).append(':').append(failure.calls)
    .append(':').append((int)storedText.charAt(1));
  });thread.start();thread.join();System.out.print(result.toString());
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
type stored struct{value *j.JavaString;caller *j.Thread;fail bool;calls int;same bool}
func(s *stored)StringJava2goExecution(e *j.Execution)*j.JavaString{s.calls++;s.same=j.ThreadCurrentThread(e)==s.caller;if s.fail{panic(j.NewIllegalStateException(""))};return s.value}
func main(){
 value:=j.NewJavaStringUTF16([]uint16{'Q',0xD800});creator:=j.ThreadCurrentThread(j.NewExecution());var result string
 thread:=j.NewThread(j.NewRunnableFuncAdapter(func(e *j.Execution){
  worker:=j.ThreadCurrentThread(e);source:=&stored{value:value,caller:worker};absent:=&stored{caller:worker};failure:=&stored{value:value,caller:worker,fail:true}
  empty:=j.JavaStringLiteralUTF16(nil);nullLiteral:=j.JavaStringLiteralUTF16([]uint16{'n','u','l','l'})
  direct:=j.JavaStringValueOfExecution(e,absent)
  nullText:=j.ConcatJavaStrings(empty,j.JavaStringTextOperandExecution(e,absent))
  storedText:=j.ConcatJavaStrings(empty,j.JavaStringTextOperandExecution(e,source))
  identity:=j.JavaStringValueOfExecution(e,source)==value
  caught:=func()(ok bool){defer func(){ok=j.CaughtAs(recover(),"IllegalStateException")}();j.ConcatJavaStrings(empty,j.JavaStringTextOperandExecution(e,failure));return}()
  ordinaryNull:=j.ConcatJavaStrings(empty,j.JavaStringTextOperandExecution(e,nil))
  result=fmt.Sprintf("%t:%t:%t:%t:%t:%t:%t:%d:%d:%d:%d",direct==nil,nullText.Equals(nullLiteral),storedText.Equals(value)&&storedText!=value,identity,ordinaryNull.Equals(nullLiteral),caught,source.same&&absent.same&&failure.same&&worker!=creator,source.calls,absent.calls,failure.calls,storedText.CharAt(1))
 }));thread.Start();thread.Join();fmt.Print(result)
}`)
}

func TestCampaignStringBoundaryStaticPrimitiveConversions(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringBoundaryPrimitives", `public class StringBoundaryPrimitives {
 public static void main(String[] args){
  StringBuilder out=new StringBuilder();
  out.append(String.valueOf(true)=="true").append(':').append(String.valueOf(false)=="false").append('|');
  char[] chars={'A',0,(char)0x4E2D,(char)0xD800,(char)0xDC00};
  for(char c:chars){String s=String.valueOf(c);out.append(s.length()).append(':').append((int)s.charAt(0)).append(';');}
  out.append('|');int[] ints={0,-1,Integer.MIN_VALUE,Integer.MAX_VALUE};for(int n:ints)out.append(String.valueOf(n)).append(';');
  out.append('|');long[] longs={0,-1,Long.MIN_VALUE,Long.MAX_VALUE};for(long n:longs)out.append(String.valueOf(n)).append(';');
  out.append('|');float[] floats={0.0f,-0.0f,Float.MIN_VALUE,Float.MAX_VALUE,Float.NaN,Float.POSITIVE_INFINITY,Float.NEGATIVE_INFINITY,(float)16777217};for(float n:floats)out.append(String.valueOf(n)).append(';');
  out.append('|');double[] doubles={0.0d,-0.0d,Double.MIN_VALUE,Double.MAX_VALUE,Double.NaN,Double.POSITIVE_INFINITY,Double.NEGATIVE_INFINITY,16777217d};for(double n:doubles)out.append(String.valueOf(n)).append(';');
  System.out.print(out.toString());
 }
}`, `package main
import("fmt";"math";"strings";j "github.com/NickyBoy89/java2go/stdjava")
func ascii(s *j.JavaString)string{var b strings.Builder;for _,u:=range s.UTF16Copy(){if u>127{panic("numeric formatting produced non-ASCII unit")};b.WriteByte(byte(u))};return b.String()}
func main(){
 var out strings.Builder
 fmt.Fprintf(&out,"%t:%t|",j.JavaStringValueOfBoolean(true)==j.JavaStringLiteralUTF16([]uint16{'t','r','u','e'}),j.JavaStringValueOfBoolean(false)==j.JavaStringLiteralUTF16([]uint16{'f','a','l','s','e'}))
 for _,c:=range []rune{'A',0,0x4E2D,0xD800,0xDC00}{s:=j.JavaStringValueOfChar(c);fmt.Fprintf(&out,"%d:%d;",s.Length(),s.CharAt(0))}
 out.WriteByte('|');for _,n:=range []int32{0,-1,math.MinInt32,math.MaxInt32}{out.WriteString(ascii(j.JavaStringValueOfInt(n)));out.WriteByte(';')}
 out.WriteByte('|');for _,n:=range []int64{0,-1,math.MinInt64,math.MaxInt64}{out.WriteString(ascii(j.JavaStringValueOfLong(n)));out.WriteByte(';')}
 out.WriteByte('|');for _,n:=range []float32{0,math.Float32frombits(0x80000000),math.SmallestNonzeroFloat32,math.MaxFloat32,float32(math.NaN()),float32(math.Inf(1)),float32(math.Inf(-1)),float32(16777217)}{out.WriteString(ascii(j.JavaStringValueOfFloat(n)));out.WriteByte(';')}
 out.WriteByte('|');for _,n:=range []float64{0,math.Copysign(0,-1),math.SmallestNonzeroFloat64,math.MaxFloat64,math.NaN(),math.Inf(1),math.Inf(-1),16777217}{out.WriteString(ascii(j.JavaStringValueOfDouble(n)));out.WriteByte(';')}
 fmt.Print(out.String())
}`)
}
