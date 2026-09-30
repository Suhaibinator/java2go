package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignStringBuilderCapacityJVMParity(t *testing.T) {
	const source = `public class CampaignBuilderCapacity {
 static int calls;
 static int capacity(){calls++;return 8;}
 static String text(){calls++;return "A😀Z";}
 static Integer boxed(){calls++;return Integer.valueOf(4);}
 static String negative(int value){try{new StringBuilder(value);return "missing";}catch(NegativeArraySizeException ex){return ex.getMessage();}}
 public static String run(){
 byte b=2;short s=3;char c='A';
 Byte boxedByte=Byte.valueOf(b);Short boxedShort=Short.valueOf(s);Character boxedChar=Character.valueOf(c);
 StringBuilder a=new StringBuilder(capacity());a.append("😀");
 StringBuilder zero=new StringBuilder(0);zero.append("z");
 StringBuilder bytes=new StringBuilder(b);StringBuilder shorts=new StringBuilder(s);StringBuilder chars=new StringBuilder(c);
 StringBuilder boxedB=new StringBuilder(boxedByte);StringBuilder boxedS=new StringBuilder(boxedShort);StringBuilder boxedC=new StringBuilder(boxedChar);StringBuilder boxedI=new StringBuilder(boxed());
 Integer absent=null;boolean nullRejected=false;try{new StringBuilder(absent);}catch(NullPointerException ex){nullRejected=true;}
 StringBuilder seeded=new StringBuilder(text());String empty=null;boolean textNull=false;try{new StringBuilder(empty);}catch(NullPointerException ex){textNull=true;}
 boolean literalNull=false;try{new StringBuilder((String)null);}catch(NullPointerException ex){literalNull=true;}
 StringBuilder fresh=new StringBuilder();
 return a.toString()+":"+a.length()+":"+zero.toString()+":"+bytes.length()+":"+shorts.length()+":"+chars.length()+":"+boxedB.length()+":"+boxedS.length()+":"+boxedC.length()+":"+boxedI.length()+":"+negative(-1)+":"+negative(-7)+":"+negative(Integer.MIN_VALUE)+":"+nullRejected+":"+textNull+":"+literalNull+":"+seeded.toString()+":"+seeded.length()+":"+fresh.length()+":"+calls;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignBuilderCapacity", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestCapacity(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
