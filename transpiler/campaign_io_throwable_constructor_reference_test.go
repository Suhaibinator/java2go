package transpiler

import (
	"fmt"
	"testing"
)

// Constructor detail messages remain Java references even when Object conversion
// or a cause callback supplies them. The original IO parity tests also exercise
// ordinary ASCII details; these controls reject host round trips and lost state.
func TestCampaignIOThrowableConstructorReferenceJVMParity(t *testing.T) {
	const source = `public class IOThrowableConstructorReference {
 static String units(String value){if(value==null)return "null";String text="";for(int i=0;i<value.length();i++)text+=(int)value.charAt(i)+",";return text;}
 static boolean initialized(Throwable value){try{value.initCause(null);return false;}catch(IllegalStateException expected){return true;}}
 public static String run(){
  String payload=new String(new char[]{'Q',(char)0xd800,0,(char)0xdc00});
  AssertionError assertion=new AssertionError((Object)payload);IllegalArgumentException illegal=new IllegalArgumentException(payload);
  AssertionError empty=new AssertionError();AssertionError absent=new AssertionError((Object)null);
  IllegalArgumentException nullMessage=new IllegalArgumentException((String)null);IllegalArgumentException nullCause=new IllegalArgumentException((Throwable)null);
  Throwable cause=new Exception("cause");AssertionError pair=new AssertionError(payload,cause);IllegalArgumentException illegalPair=new IllegalArgumentException(payload,cause);
  return (assertion.getMessage()==payload)+":"+(illegal.getMessage()==payload)+":"+units(assertion.getMessage())+":"+units(illegal.getMessage())+":"+units(empty.getMessage())+":"+units(absent.getMessage())+":"+initialized(empty)+":"+initialized(absent)+":"+initialized(nullMessage)+":"+initialized(nullCause)+":"+(pair.getMessage()==payload)+":"+(pair.getCause()==cause)+":"+initialized(pair)+":"+(illegalPair.getMessage()==payload)+":"+(illegalPair.getCause()==cause)+":"+initialized(illegalPair);
 }
}`
	ioThrowableConstructorReferenceParity(t, "IOThrowableConstructorReference", source)
}

func TestCampaignIOThrowableConstructorCallbackJVMParity(t *testing.T) {
	const source = `public class IOThrowableConstructorCallback {
 static class Detail {String text;int calls;boolean held;Detail(String value){text=value;}public String toString(){calls++;held=Thread.holdsLock(this);return text;}}
 static class Cause extends RuntimeException {String text;int calls;boolean held;Cause(String value){text=value;}public String toString(){calls++;held=Thread.holdsLock(this);return text;}}
 public static String run(){
  String payload=new String(new char[]{(char)0xd800,0,(char)0xdc00});Detail detail=new Detail(payload);AssertionError assertion;
  synchronized(detail){assertion=new AssertionError((Object)detail);}
  Cause cause=new Cause(payload);AssertionError fromCause;IllegalArgumentException illegal;
  synchronized(cause){fromCause=new AssertionError((Object)cause);illegal=new IllegalArgumentException((Throwable)cause);}
  detail.text=null;AssertionError nullText; synchronized(detail){nullText=new AssertionError((Object)detail);}
  return (assertion.getMessage()==payload)+":"+detail.calls+":"+detail.held+":"+(fromCause.getMessage()==payload)+":"+(fromCause.getCause()==cause)+":"+(illegal.getMessage()==payload)+":"+(illegal.getCause()==cause)+":"+cause.calls+":"+cause.held+":"+(nullText.getMessage()==null);
 }
}`
	ioThrowableConstructorReferenceParity(t, "IOThrowableConstructorCallback", source)
}

func TestCampaignIOAssertionPrimitiveConstructorsJVMParity(t *testing.T) {
	const source = `public class IOAssertionPrimitiveConstructors {
 public static String run(){AssertionError unit=new AssertionError((char)0xd800);return unit.getMessage().length()+":"+(int)unit.getMessage().charAt(0)+":"+new AssertionError(true).getMessage()+":"+new AssertionError((byte)-2).getMessage()+":"+new AssertionError((short)3).getMessage()+":"+new AssertionError(4).getMessage()+":"+new AssertionError(5L).getMessage()+":"+new AssertionError(-0.0f).getMessage()+":"+new AssertionError(1.25).getMessage();}
}`
	ioThrowableConstructorReferenceParity(t, "IOAssertionPrimitiveConstructors", source)
}

func ioThrowableConstructorReferenceParity(t *testing.T, class, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, class, source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("slices";"testing";"unicode/utf16";j "github.com/NickyBoy89/java2go/stdjava")
func TestConstructorReference(t *testing.T){var got *j.JavaString=Run();if got==nil{t.Fatal("Run returned null")};want:=utf16.Encode([]rune(%q));if units:=got.UTF16Copy();!slices.Equal(units,want){t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x",want,units)}}`, want))
}
