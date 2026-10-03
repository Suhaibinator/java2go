package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func campaignAssertionErrorOracle(t *testing.T, name, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, name, source)
	t.Logf("JVM AssertionError oracle: %q", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestAssertionError(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != Go UTF16 %%#v", %q, units) }
}`, utf16.Encode([]rune(want)), want))
}

func TestCampaignRuntimeAssertionErrorConstructors(t *testing.T) {
	const source = `public class CampaignAssertionErrorConstructors {
 static String state(AssertionError error){boolean initialized=false;try{error.initCause(null);}catch(IllegalStateException expected){initialized=true;}return (error.getMessage()==null?"<null>":error.getMessage())+":"+initialized;}
 public static String run(){
  Exception cause=new Exception("boom");AssertionError fromCause=new AssertionError(cause);boolean identity=fromCause.getCause()==cause;
  AssertionError pair=new AssertionError("pair",cause);boolean pairIdentity=pair.getCause()==cause;
  String absent=null;Character boxed=Character.valueOf('B');Integer number=Integer.valueOf(7);
  return state(new AssertionError())+"|"+state(new AssertionError((Object)null))+"|"+state(new AssertionError(absent))+"|"+state(new AssertionError("text"))+"|"+state(fromCause)+":"+identity+"|"+state(pair)+":"+pairIdentity+"|"+state(new AssertionError(null,null))+"|"+state(new AssertionError(true))+"|"+state(new AssertionError('A'))+"|"+state(new AssertionError((byte)-2))+"|"+state(new AssertionError((short)3))+"|"+state(new AssertionError(4))+"|"+state(new AssertionError(5L))+"|"+state(new AssertionError(-0.0f))+"|"+state(new AssertionError(1.25))+"|"+state(new AssertionError(boxed))+"|"+state(new AssertionError(number));
 }
}`
	campaignAssertionErrorOracle(t, "CampaignAssertionErrorConstructors", source)
}

func TestCampaignRuntimeAssertionErrorObjectExecution(t *testing.T) {
	const source = `public class CampaignAssertionErrorObjectExecution {
 static Thread caller;static int calls=0;static int arguments=0;static int cleanup=0;
 static Object detail(Object value){arguments++;return value;}
 static class Detail {public synchronized String toString(){calls++;return "detail:"+Thread.holdsLock(this)+":"+(Thread.currentThread()==caller);}}
 static class Cause extends Exception {public String toString(){calls++;return "cause:"+(Thread.currentThread()==caller);}}
 static class NullText {public String toString(){calls++;return null;}}
 static class Broken {public String toString(){calls++;throw new IllegalArgumentException("conversion");}}
 public static String run(){caller=Thread.currentThread();Detail value=new Detail();String message;
  synchronized(value){message=new AssertionError(detail(value)).getMessage();}
  Cause cause=new Cause();AssertionError error=new AssertionError(detail(cause));boolean same=error.getCause()==cause;
  boolean nullText=new AssertionError(detail(new NullText())).getMessage()==null;
  String failure="none";try{new AssertionError(detail(new Broken()));}catch(IllegalArgumentException expected){failure=expected.getMessage();}finally{cleanup++;}
  return message+"|"+error.getMessage()+":"+same+"|"+nullText+"|"+failure+"|"+calls+":"+arguments+":"+cleanup;
 }
}`
	campaignAssertionErrorOracle(t, "CampaignAssertionErrorObjectExecution", source)
}

func TestCampaignRuntimeAssertionErrorSubclassConstructors(t *testing.T) {
	const source = `public class CampaignAssertionErrorSubclassConstructors {
 static int arguments=0;
 static Object detail(Object value){arguments++;return value;}
 static class Empty extends AssertionError {Empty(){super();}}
 static class Implicit extends AssertionError {}
 static class FromObject extends AssertionError {FromObject(Object value){super(value);}}
 static class FromChar extends AssertionError {FromChar(char value){super(value);}}
 static class FromPair extends AssertionError {FromPair(String message,Throwable cause){super(message,cause);}}
 static String state(Throwable error){boolean initialized=false;try{error.initCause(null);}catch(IllegalStateException expected){initialized=true;}return (error.getMessage()==null?"<null>":error.getMessage())+":"+initialized;}
 public static String run(){Exception cause=new Exception("sub");FromObject error=new FromObject(detail(cause));FromPair pair=new FromPair("pair",cause);boolean objectCause=error.getCause()==cause;boolean pairCause=pair.getCause()==cause;
 return state(new Empty())+"|"+state(new Implicit())+"|"+state(error)+":"+objectCause+"|"+state(new FromObject(detail(null)))+"|"+state(new FromChar('Q'))+"|"+state(pair)+":"+pairCause+"|"+state(new FromPair(null,null))+"|"+arguments;
 }
}`
	campaignAssertionErrorOracle(t, "CampaignAssertionErrorSubclassConstructors", source)
}
