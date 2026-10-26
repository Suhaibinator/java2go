package transpiler

import (
	"fmt"
	"testing"
)

// Every expected observation is captured from the JVM before either runtime
// driver is built. Diagnostics are serialized as UTF16 units, so replacement
// characters cannot hide an isolated surrogate crossing the String boundary.
func TestCampaignBigMathCanonicalUTF16ParsingJDK21(t *testing.T) {
	campaignStringCoreOperationOracle(t, "BigMathCanonicalUTF16", `import java.math.BigInteger;import java.math.BigDecimal;
public class BigMathCanonicalUTF16 {
 static String units(String text){if(text==null)return "null";StringBuilder out=new StringBuilder();for(int i=0;i<text.length();i++)out.append((int)text.charAt(i)).append(',');return out.toString();}
 static void record(boolean decimal,String text){try{if(decimal){BigDecimal value=new BigDecimal(text);System.out.print("ok:"+units(value.toString())+":"+value.scale());}else System.out.print("ok:"+units(new BigInteger(text).toString()));}catch(NullPointerException absent){System.out.print("NullPointerException");}catch(NumberFormatException failure){Throwable cause=failure.getCause();boolean initialized=false;try{failure.initCause(null);}catch(IllegalStateException already){initialized=true;}System.out.print("NumberFormatException:"+units(failure.getMessage())+":"+(cause==null?"none":cause.getClass().getSimpleName()+":"+units(cause.getMessage()))+":"+initialized);}System.out.println();}
 public static void main(String[] args){String[] inputs={null,"","+","-","-+1","1-2","+00017","-000","184467440737095516161234567890","\u0661\u0662\uff13","+00017.00","-0.000","0E+7","1.2300e+5",".5","5.","\u0661.\uff12E\u0662","1E+2147483648","1E-2147483648","1.0E-2147483647",".","1.2.3","1e","1e+","1eX","1e99999999999"," 1","NaN","1234567890123x567890",new String(new char[]{(char)0xd800}),new String(new char[]{(char)0xdc00}),new String(new char[]{'1',(char)0xd800,'2'}),new String(new char[]{'1',(char)0xd835,(char)0xdfd9,'2'}),new String(new char[]{'1',0,'2'}),new String(new char[]{'0','0','1','2','3','4','5','6','7','8','9',(char)0xdc00,'1','2','3','4','5','6','7','8','9'}),new String(new char[]{'1','2','3','4','5','6','7','8','9','0','1','2','3','4','5','6','7','8',(char)0xd800})};for(String input:inputs){record(false,input);record(true,input);}}
}`, `package main
import("fmt";"strings";j "github.com/NickyBoy89/java2go/stdjava")
func units(text *j.JavaString)string{if text==nil{return "null"};var out strings.Builder;for _,u:=range text.UTF16Copy(){fmt.Fprintf(&out,"%d,",u)};return out.String()}
func record(decimal bool,text *j.JavaString){defer func(){if failure:=recover();failure!=nil{if j.CaughtAs(failure,"NullPointerException"){fmt.Print("NullPointerException")}else if j.CaughtAs(failure,"NumberFormatException"){cause:=j.GetCause(failure);initialized:=func()(bad bool){defer func(){bad=j.CaughtAs(recover(),"IllegalStateException")}();j.ThrowableInitCauseExecution(j.NewExecution(),failure,nil);return}();observed:="none";if cause!=nil{observed=j.ObjectGetClass(cause).GetSimpleName()+":"+units(j.JavaThrowableMessageDefault(cause))};fmt.Printf("NumberFormatException:%s:%s:%t",units(j.JavaThrowableMessageDefault(failure)),observed,initialized)}else{panic(failure)}};fmt.Println()}();if decimal{value:=j.NewBigDecimalJavaString(text);fmt.Printf("ok:%s:%d",units(value.StringJava2goExecution(j.NewExecution())),value.Scale())}else{fmt.Printf("ok:%s",units(j.NewBigIntegerJavaString(text).StringJava2goExecution(j.NewExecution())))}}
func text(s string)*j.JavaString{return j.JavaStringFromHostUTF8(s)}
func main(){inputs:=[]*j.JavaString{nil,text(""),text("+"),text("-"),text("-+1"),text("1-2"),text("+00017"),text("-000"),text("184467440737095516161234567890"),text("\u0661\u0662\uff13"),text("+00017.00"),text("-0.000"),text("0E+7"),text("1.2300e+5"),text(".5"),text("5."),text("\u0661.\uff12E\u0662"),text("1E+2147483648"),text("1E-2147483648"),text("1.0E-2147483647"),text("."),text("1.2.3"),text("1e"),text("1e+"),text("1eX"),text("1e99999999999"),text(" 1"),text("NaN"),text("1234567890123x567890"),j.NewJavaStringUTF16([]uint16{0xd800}),j.NewJavaStringUTF16([]uint16{0xdc00}),j.NewJavaStringUTF16([]uint16{'1',0xd800,'2'}),j.NewJavaStringUTF16([]uint16{'1',0xd835,0xdfd9,'2'}),j.NewJavaStringUTF16([]uint16{'1',0,'2'}),j.NewJavaStringUTF16([]uint16{'0','0','1','2','3','4','5','6','7','8','9',0xdc00,'1','2','3','4','5','6','7','8','9'}),j.NewJavaStringUTF16([]uint16{'1','2','3','4','5','6','7','8','9','0','1','2','3','4','5','6','7','8',0xd800})};for _,input:=range inputs{record(false,input);record(true,input)}}
`)
}

func TestCampaignBigMathCanonicalGeneratedJDK21(t *testing.T) {
	const source = `import java.math.BigInteger;import java.math.BigDecimal;
public class BigMathCanonicalGenerated {
 static int arguments;static int cleanup;
 static String argument(String value){arguments++;return value;}
 static <T extends Number> String render(T value){return value.toString();}
 static String units(String text){if(text==null)return "null";StringBuilder out=new StringBuilder();for(int i=0;i<text.length();i++)out.append((int)text.charAt(i)).append(',');return out.toString();}
 static String parse(boolean decimal,String text){try{if(decimal)return new BigDecimal(argument(text)).toString();return new BigInteger(argument(text)).toString();}catch(NumberFormatException failure){return failure.getClass().getSimpleName()+":"+units(failure.getMessage());}catch(NullPointerException absent){return "NullPointerException";}finally{cleanup++;}}
 public static String run(){BigInteger integer=new BigInteger("+00017");BigDecimal decimal=new BigDecimal("17.00");Number number=integer;Object object=decimal;String integerText=integer.toString();String decimalText=decimal.toString();String[] storage={integerText,decimalText};String observed=storage[0]+":"+storage[1]+":"+number.toString()+":"+String.valueOf(object)+":"+render(integer)+":"+(""+decimal)+":"+(String.valueOf((Object)integerText)==integerText)+":"+integerText.length()+":"+(int)decimalText.charAt(2);String high=new String(new char[]{'1',(char)0xd800,'2'});String low=new String(new char[]{'1',(char)0xdc00,'2'});observed=observed+"|"+parse(false,high)+"|"+parse(true,low)+"|"+parse(false,null)+"|"+parse(true,"1e+")+"|"+parse(true,"\u0661.\uff12E\u0662");boolean nullReceiver=false;try{BigInteger absent=null;absent.toString();}catch(NullPointerException expected){nullReceiver=true;}return observed+"#"+arguments+":"+cleanup+":"+nullReceiver;}
}`
	want := campaignRuntimeJavaOracle(t, "BigMathCanonicalGenerated", source)
	t.Logf("JVM canonical BigMath oracle: %q", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestCanonicalBigMath(t *testing.T){value:=Run();encoded:=j.JavaStringGetBytes(value,j.UTF_8).Elements;bytes:=make([]byte,len(encoded));for i,b:=range encoded{bytes[i]=byte(b)};if got:=string(bytes);got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
