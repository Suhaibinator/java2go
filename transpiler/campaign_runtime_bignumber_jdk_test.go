package transpiler

import (
	"fmt"
	"testing"
)

// These programs exercise JDK services used by the complete locked Gson source.
// Expected observations come exclusively from the JDK21 oracle at execution.
func campaignBigNumberOracle(t *testing.T, name, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, name, source)
	t.Logf("JVM big-number oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestBigNumber(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}

func TestCampaignRuntimeBigNumberIdentityAndNumberViews(t *testing.T) {
	const source = `import java.math.BigInteger;import java.math.BigDecimal;
public class CampaignBigNumberIdentity {
 static <T extends Number> T same(T value){return value;}
 public static String run(){BigInteger integer=new BigInteger("17");BigDecimal decimal=new BigDecimal("17.00");Number number=integer;Object object=number;Number[] values=new Number[]{integer,decimal,null};BigInteger other=new BigInteger("17");BigDecimal copied=new BigDecimal("17.00");
  return object.getClass().getName()+":"+values[1].getClass().getName()+":"+(object instanceof BigInteger)+":"+(object instanceof BigDecimal)+":"+(object==integer)+":"+(same(integer)==integer)+":"+(values[2]==null)+":"+(integer==other)+":"+integer.equals(other)+":"+(decimal==copied)+":"+decimal.equals(copied)+":"+(BigInteger.valueOf(7)==BigInteger.valueOf(7))+":"+(BigInteger.valueOf(-16)==BigInteger.valueOf(-16))+":"+number.intValue()+":"+values[1].toString();
 }
}`
	campaignBigNumberOracle(t, "CampaignBigNumberIdentity", source)
}

func TestCampaignRuntimeBigIntegerParsing(t *testing.T) {
	const source = `import java.math.BigInteger;
public class CampaignBigIntegerParsing {
 static String parse(String text){try{return new BigInteger(text).toString();}catch(NumberFormatException bad){return bad.getClass().getSimpleName()+":"+bad.getMessage();}catch(NullPointerException absent){return "NullPointerException";}}
 public static String run(){return parse("+00017")+"|"+parse("-000")+"|"+parse("184467440737095516161234567890")+"|"+parse("\u0661\u0662\uff13")+"|"+parse("")+"|"+parse("+")+"|"+parse("-+1")+"|"+parse("1-2")+"|"+parse(" 12")+"|"+parse("1.0")+"|"+parse("\uD835\uDFD9")+"|"+parse(null);}
}`
	campaignBigNumberOracle(t, "CampaignBigIntegerParsing", source)
}

func TestCampaignRuntimeBigDecimalParsingAndScale(t *testing.T) {
	const source = `import java.math.BigDecimal;
public class CampaignBigDecimalParsing {
 static String parse(String text){try{BigDecimal value=new BigDecimal(text);return value.toString()+":"+value.scale();}catch(NumberFormatException bad){return bad.getClass().getSimpleName()+":"+bad.getMessage();}catch(NullPointerException absent){return "NullPointerException";}}
 public static String run(){return parse("+00017.00")+"|"+parse("-0.000")+"|"+parse("0E+7")+"|"+parse("1.2300e+5")+"|"+parse("0.000001")+"|"+parse("0.0000001")+"|"+parse(".5")+"|"+parse("5.")+"|"+parse("\u0661.\uff12E\u0662")+"|"+parse("1E+2147483648")+"|"+parse("1E-2147483648")+"|"+parse("1.0E-2147483647")+"|"+parse("")+"|"+parse(".")+"|"+parse("1.2.3")+"|"+parse("1e+")+"|"+parse("1e99999999999")+"|"+parse(" 1")+"|"+parse("NaN")+"|"+parse(null);}
}`
	campaignBigNumberOracle(t, "CampaignBigDecimalParsing", source)
}

func TestCampaignRuntimeBigNumberComparisonAndHash(t *testing.T) {
	const source = `import java.math.BigInteger;import java.math.BigDecimal;
public class CampaignBigNumberComparison {
 public static String run(){BigDecimal first=new BigDecimal("2.0");BigDecimal scaled=new BigDecimal("2.00");BigDecimal other=new BigDecimal("-2.00");BigDecimal zero=new BigDecimal("-0.0");BigDecimal scaleZero=new BigDecimal("0.00");BigInteger huge=new BigInteger("18446744073709551617");BigInteger negative=new BigInteger("-18446744073709551617");boolean absent=false;try{first.compareTo(null);}catch(NullPointerException expected){absent=true;}
  return first.equals(scaled)+":"+first.compareTo(scaled)+":"+first.hashCode()+":"+scaled.hashCode()+":"+other.hashCode()+":"+zero.equals(scaleZero)+":"+zero.compareTo(scaleZero)+":"+zero.hashCode()+":"+scaleZero.hashCode()+":"+huge.hashCode()+":"+negative.hashCode()+":"+huge.compareTo(negative)+":"+huge.equals(new BigInteger("18446744073709551617"))+":"+huge.equals(first)+":"+first.equals(null)+":"+absent;
 }
}`
	campaignBigNumberOracle(t, "CampaignBigNumberComparison", source)
}

func TestCampaignRuntimeBigNumberConversions(t *testing.T) {
	const source = `import java.math.BigInteger;import java.math.BigDecimal;
public class CampaignBigNumberConversions {
 static String convert(Number value){return value.byteValue()+":"+value.shortValue()+":"+value.intValue()+":"+value.longValue()+":"+Float.floatToRawIntBits(value.floatValue())+":"+Double.doubleToRawLongBits(value.doubleValue());}
 public static String run(){return convert(new BigInteger("18446744073709551617"))+"|"+convert(new BigInteger("-18446744073709551617"))+"|"+convert(new BigInteger("9007199254740993"))+"|"+convert(new BigDecimal("4294967297.99"))+"|"+convert(new BigDecimal("-4294967297.99"))+"|"+convert(new BigDecimal("1.000000059604644775390625"))+"|"+convert(new BigDecimal("1.000000059604644775390626"))+"|"+convert(new BigDecimal("1E+400"))+"|"+convert(new BigDecimal("-1E-400"))+"|"+convert(new BigDecimal("1E-45"))+"|"+convert(new BigDecimal("1E+2147483647"))+"|"+convert(new BigDecimal("1E-2147483647"));}
}`
	campaignBigNumberOracle(t, "CampaignBigNumberConversions", source)
}

func TestCampaignRuntimeBigNumberParseCausesAndCleanup(t *testing.T) {
	const source = `import java.math.BigInteger;import java.math.BigDecimal;
public class CampaignBigNumberCauses {
 static int arguments=0;static int cleanup=0;
 static String argument(String value){arguments++;return value;}
 static String parse(boolean decimal,String text){try{if(decimal)return new BigDecimal(argument(text)).toString();return new BigInteger(argument(text)).toString();}catch(RuntimeException failure){Throwable cause=failure.getCause();String observed=cause==null?"none":cause.getClass().getSimpleName()+":"+cause.getMessage();boolean initialized=false;try{failure.initCause(null);}catch(IllegalStateException already){initialized=true;}return failure.getClass().getSimpleName()+"/"+observed+"/"+initialized;}finally{cleanup++;}}
 public static String run(){String observed=parse(true,"")+"|"+parse(true,"1e")+"|"+parse(true,"1e+")+"|"+parse(true,".")+"|"+parse(true,"1eX")+"|"+parse(false,"")+"|"+parse(false,"1.0")+"|"+parse(true,null);return observed+"#"+arguments+":"+cleanup;}
}`
	campaignBigNumberOracle(t, "CampaignBigNumberCauses", source)
}
