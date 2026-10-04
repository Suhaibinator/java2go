package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignBigMathCanonicalIdentityJDK21(t *testing.T) {
	const source = `import java.math.BigInteger;import java.math.BigDecimal;
public class BigMathCanonicalIdentity {
 public static String run(){BigInteger zero=new BigInteger("0"),negativeZero=new BigInteger("-000"),integer=new BigInteger("17");String zeroText=zero.toString(),integerText=integer.toString(),integerAgain=integer.toString();BigDecimal decimal=new BigDecimal("17.00"),equal=new BigDecimal("17.00");String decimalText=decimal.toString(),decimalAgain=decimal.toString(),equalText=equal.toString();Number number=decimal;Object object=decimal;return (zeroText=="0")+":"+(negativeZero.toString()=="0")+":"+(BigInteger.valueOf(0).toString()=="0")+":"+(String.valueOf((Object)zero)==zeroText)+":"+(integerText!=integerAgain)+":"+integerText.equals(integerAgain)+":"+(String.valueOf((Object)integer)!=integerText)+":"+(decimalText==decimalAgain)+":"+(number.toString()==decimalText)+":"+(String.valueOf(object)==decimalText)+":"+(decimalText!=equalText)+":"+decimalText.equals(equalText)+":"+(equal.toString()==equalText);}
 public static void main(String[] args){System.out.print(run());}
}`
	campaignStringCoreOperationOracle(t, "BigMathCanonicalIdentity", source, `package main
import("fmt";"sync";j "github.com/NickyBoy89/java2go/stdjava")
func text(s string)*j.JavaString{return j.JavaStringFromHostUTF8(s)}
func main(){e:=j.NewExecution();zero:=j.NewBigIntegerJavaString(text("0"));negativeZero:=j.NewBigIntegerJavaString(text("-000"));integer:=j.NewBigIntegerJavaString(text("17"));zeroText:=zero.StringJava2goExecution(e);integerText:=integer.StringJava2goExecution(e);integerAgain:=integer.StringJava2goExecution(e);decimal:=j.NewBigDecimalJavaString(text("17.00"));equal:=j.NewBigDecimalJavaString(text("17.00"));decimalText:=decimal.StringJava2goExecution(e);decimalAgain:=decimal.StringJava2goExecution(e);equalText:=equal.StringJava2goExecution(e);var number j.JavaNumber=decimal;var object any=decimal;fmt.Printf("%t:%t:%t:%t:%t:%t:%t:%t:%t:%t:%t:%t:%t",zeroText==j.JavaStringLiteralUTF16([]uint16{'0'}),negativeZero.StringJava2goExecution(e)==j.JavaStringLiteralUTF16([]uint16{'0'}),j.BigIntegerValueOf(0).StringJava2goExecution(e)==j.JavaStringLiteralUTF16([]uint16{'0'}),j.JavaStringValueOfExecution(e,zero)==zeroText,integerText!=integerAgain,integerText.Equals(integerAgain),j.JavaStringValueOfExecution(e,integer)!=integerText,decimalText==decimalAgain,j.JavaStringValueOfExecution(e,number)==decimalText,j.JavaStringValueOfExecution(e,object)==decimalText,decimalText!=equalText,decimalText.Equals(equalText),equal.StringJava2goExecution(e)==equalText)
 if decimalAgain!=decimalText{return}
 // The existing cache is immutable across concurrent Java-facing reads.
 var workers sync.WaitGroup;for n:=0;n<8;n++{workers.Go(func(){for i:=0;i<100;i++{if decimal.StringJava2goExecution(j.NewExecution())!=decimalText{panic("cached BigDecimal String identity changed")}}})};workers.Wait()
 // Exercise first publication concurrently on an independently allocated value.
 fresh:=j.NewBigDecimalJavaString(text("123.45"));results:=make([]*j.JavaString,8);for n:=range results{workers.Go(func(){results[n]=fresh.StringJava2goExecution(j.NewExecution())})};workers.Wait();for _,result:=range results{if !result.Equals(fresh.StringJava2goExecution(e)){panic("BigDecimal String cache changed numeric content")}};published:=fresh.StringJava2goExecution(e);if fresh.StringJava2goExecution(e)!=published{panic("BigDecimal String identity changed after publication")}
}
`)
	want := campaignRuntimeJavaOracle(t, "BigMathCanonicalIdentity", source)
	t.Logf("JVM canonical BigMath identity oracle: %q", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestCanonicalIdentity(t *testing.T){value:=Run();encoded:=j.JavaStringGetBytes(value,j.UTF_8).Elements;bytes:=make([]byte,len(encoded));for i,b:=range encoded{bytes[i]=byte(b)};if got:=string(bytes);got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
