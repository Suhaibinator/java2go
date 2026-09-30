package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignMathMultiplyExactJVMParity(t *testing.T) {
	const source = `public class CampaignMathMultiplyExact{
 static String trace="";
 static int value(String name,int value){trace+=name;return value;}
 static String integer(int a,int b){try{return "ok="+Math.multiplyExact(a,b);}catch(ArithmeticException ex){return ex.getMessage();}}
 static String wide(long a,long b){try{return "ok="+Math.multiplyExact(a,b);}catch(ArithmeticException ex){return ex.getMessage();}}
 public static String run(){
 int nested=Math.addExact(value("a",5),Math.multiplyExact(value("b",7),value("c",6)));
 try{Math.addExact(value("d",1),Math.multiplyExact(value("e",2147483647),value("f",2)));}catch(ArithmeticException ex){trace+="!";}
 return nested+":"+trace+":"+integer(2147483647,1)+":"+integer(2147483647,2)+":"+integer(-2147483648,-1)+":"+integer(-46341,46341)+":"+integer(0,-2147483648)+":"+wide(9223372036854775807L,2L)+":"+wide(Long.MIN_VALUE,-1L)+":"+wide(-1L,Long.MIN_VALUE)+":"+wide(Long.MIN_VALUE,1L)+":"+Math.multiplyExact(2147483647L,2)+":"+Math.multiplyExact(2,2147483647L);
 }} `
	want := campaignRuntimeJavaOracle(t, "CampaignMathMultiplyExact", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestMultiply(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
