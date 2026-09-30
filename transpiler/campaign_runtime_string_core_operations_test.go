package transpiler

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This checks runtime operations directly against a JVM oracle. The handwritten
// Go driver is test code, not translated application output or a source-library
// substitute. Compiler expression/override/constant lowering has separate gates.
func campaignStringCoreOperationOracle(t *testing.T, class, javaSource, goDriver string) {
	t.Helper()
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	javaPath := filepath.Join(temporary, class+".java")
	if err := os.WriteFile(javaPath, []byte(javaSource), 0600); err != nil {
		t.Fatal(err)
	}
	campaignStringRequireSuccess(t, "operation javac", campaignStringBoundedRun(t, "operation javac", temporary, javac, 300*time.Second, "--release", "21", "-encoding", "UTF-8", "-d", temporary, javaPath))
	want := campaignStringBoundedRun(t, "operation JVM", temporary, java, 60*time.Second, "-cp", temporary, class)
	campaignStringRequireSuccess(t, "operation JVM", want)
	if len(want.stderr) != 0 {
		t.Fatalf("unexpected JVM stderr: %q", want.stderr)
	}
	t.Logf("JVM operation oracle=%q", want.stdout)
	module := "module operationprobe\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(temporary, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temporary, "main.go"), []byte(goDriver), 0600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(temporary, "operations")
	campaignStringRequireSuccess(t, "operation runtime driver build", campaignStringBoundedRun(t, "operation runtime driver build", temporary, "go", 300*time.Second, "build", "-race", "-mod=mod", "-o", executable, "."))
	got := campaignStringBoundedRun(t, "operation runtime driver", temporary, executable, 60*time.Second)
	if got.exit != want.exit || !bytes.Equal(got.stdout, want.stdout) || !bytes.Equal(got.stderr, want.stderr) {
		t.Fatalf("JVM exit%d stdout%q stderr%q; runtime exit%d stdout%q stderr%q", want.exit, want.stdout, want.stderr, got.exit, got.stdout, got.stderr)
	}
}

func TestCampaignStringCoreOperationAllocation(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringOperationAllocation", `public class StringOperationAllocation {
 public static void main(String[] args) {
  String value = new String("ab");
  String empty = new String("");
  String first = value + empty;
  String second = value + empty;
  StringBuilder builder = new StringBuilder().append('a').append('b');
  String a = builder.toString(), b = builder.toString();
  builder.append('c');
  System.out.print((value.substring(0, value.length()) == value) + ":" + (value.concat(empty) == value)
   + ":" + (first != value && first != second && first.equals(value))
   + ":" + (a != b && a.equals(b)) + ":" + a.length() + ":" + builder.length());
 }
}`, `package main
import("fmt"; j "github.com/NickyBoy89/java2go/stdjava")
func main() {
 value:=j.NewJavaStringUTF16([]uint16{'a','b'})
 empty:=j.NewJavaStringUTF16(nil)
 first,second:=j.ConcatJavaStrings(value,empty),j.ConcatJavaStrings(value,empty)
 builder:=j.NewStringBuilder().Append(rune('a')).Append(rune('b'))
 a,b:=builder.ToJavaString(),builder.ToJavaString()
 builder.Append(rune('c'))
 fmt.Printf("%t:%t:%t:%t:%d:%d",value.Substring(0,value.Length())==value,value.Concat(empty)==value,first!=value&&first!=second&&first.Equals(value),a!=b&&a.Equals(b),a.Length(),builder.Length())
}`)
}

func TestCampaignStringCoreOperationSurrogates(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringOperationSurrogates", `public class StringOperationSurrogates {
 public static void main(String[] args) {
  char[] units = {'Q', (char)0xD83D, (char)0xDE03, 'Z'};
  String value = new String(units);
  units[0]='X';
  char[] exposed=value.toCharArray(); exposed[3]='X';
  String high=value.substring(1,2), low=value.substring(2,3);
  StringBuilder builder=new StringBuilder().append((char)0xD800);
  String one=builder.toString(), two=builder.toString();
  builder.append('A');
  System.out.print(value.length()+":"+(int)value.charAt(0)+":"+(int)value.charAt(3)
   +":"+high.length()+":"+(int)high.charAt(0)+":"+low.length()+":"+(int)low.charAt(0)
   +":"+(one!=two&&one.equals(two))+":"+one.length()+":"+(int)one.charAt(0)+":"+builder.length());
 }
}`, `package main
import("fmt"; j "github.com/NickyBoy89/java2go/stdjava")
func main() {
 units:=[]uint16{'Q',0xD83D,0xDE03,'Z'}
 value:=j.NewJavaStringUTF16(units);units[0]='X'
 exposed:=value.UTF16Copy();exposed[3]='X'
 high,low:=value.Substring(1,2),value.Substring(2,3)
 builder:=j.NewStringBuilder().Append(rune(0xD800))
 one,two:=builder.ToJavaString(),builder.ToJavaString();builder.Append(rune('A'))
 fmt.Printf("%d:%d:%d:%d:%d:%d:%d:%t:%d:%d:%d",value.Length(),value.CharAt(0),value.CharAt(3),high.Length(),high.CharAt(0),low.Length(),low.CharAt(0),one!=two&&one.Equals(two),one.Length(),one.CharAt(0),builder.Length())
}`)
}

func TestCampaignStringCoreOperationValueOfExecution(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringOperationValueOf", `public class StringOperationValueOf {
 static final class Stored {
  final String value; final Thread caller; final boolean fail; int calls; boolean same;
  Stored(String value, Thread caller, boolean fail) {this.value=value;this.caller=caller;this.fail=fail;}
  public String toString() {calls++;same=Thread.currentThread()==caller;if(fail)throw new IllegalStateException();return value;}
 }
 public static void main(String[] args) throws Exception {
  String value=new String("stored");
  StringBuilder output=new StringBuilder();
  Thread creator=Thread.currentThread();
  Thread worker=new Thread(()-> {
   Stored stored=new Stored(value,Thread.currentThread(),false);
   Stored absent=new Stored(null,Thread.currentThread(),false);
   Stored failure=new Stored(value,Thread.currentThread(),true);
   boolean identity=String.valueOf((Object)stored)==value;
   boolean nullResult=String.valueOf((Object)absent)==null;
   boolean caught=false;
   try {String.valueOf((Object)failure);} catch(IllegalStateException expected) {caught=true;}
   output.append(identity).append(':').append(nullResult).append(':').append(String.valueOf((Object)value)==value)
    .append(':').append(String.valueOf((Object)null).equals("null"))
    .append(':').append(String.valueOf((String)null).equals("null"))
    .append(':').append(stored.same&&absent.same&&failure.same&&Thread.currentThread()!=creator)
    .append(':').append(caught).append(':').append(stored.calls+absent.calls+failure.calls);
  });worker.start();worker.join();System.out.print(output.toString());
 }
}`, `package main
import("fmt"; j "github.com/NickyBoy89/java2go/stdjava")
type stored struct {value *j.JavaString; caller *j.Thread; fail bool; calls int; same bool}
func(s *stored)StringJava2goExecution(e *j.Execution)*j.JavaString {
 s.calls++;s.same=j.ThreadCurrentThread(e)==s.caller
 if s.fail {panic(j.NewIllegalStateException(""))};return s.value
}
func main() {
 value:=j.NewJavaStringUTF16([]uint16{'s','t','o','r','e','d'})
 creator:=j.NewExecution()
 mainThread:=j.ThreadCurrentThread(creator)
 var result string
 thread:=j.NewThread(j.NewRunnableFuncAdapter(func(worker *j.Execution) {
 workerThread:=j.ThreadCurrentThread(worker)
 source:=&stored{value:value,caller:workerThread};absent:=&stored{caller:workerThread};failure:=&stored{value:value,caller:workerThread,fail:true}
 identity:=j.JavaStringValueOfExecution(worker,source)==value
 nullResult:=j.JavaStringValueOfExecution(worker,absent)==nil
 caught:=func()(ok bool){defer func(){ok=j.CaughtAs(recover(),"IllegalStateException")}();j.JavaStringValueOfExecution(worker,failure);return}()
 nullText:=j.JavaStringLiteralUTF16([]uint16{'n','u','l','l'})
 var typedNull *j.JavaString
 result=fmt.Sprintf("%t:%t:%t:%t:%t:%t:%t:%d",identity,nullResult,j.JavaStringValueOfExecution(worker,value)==value,j.JavaStringValueOfExecution(worker,nil).Equals(nullText),j.JavaStringValueOfExecution(worker,typedNull).Equals(nullText),source.same&&absent.same&&failure.same&&workerThread!=mainThread,caught,source.calls+absent.calls+failure.calls)
 }))
 thread.Start();thread.Join()
 fmt.Print(result)
}`)
}

// Additive checked-message coverage, prepared from JDK21 String.checkBoundsBeginEnd
// and Preconditions.checkFromToIndex. Null assertions are type-only: helpful
// JVM null messages depend on the call site's bytecode, not just this helper.
func TestCampaignStringCoreOperationBounds(t *testing.T) {
	campaignStringCoreOperationOracle(t, "StringOperationBounds", `public class StringOperationBounds {
 public static void main(String[] args) {
  String value=new String("ab"); StringBuilder out=new StringBuilder();
  int[][] ranges={{-1,1},{0,3},{2,1}};
  for(int[] range:ranges) {
   try {value.substring(range[0],range[1]);out.append("missing");}
   catch(StringIndexOutOfBoundsException expected){out.append(expected.getMessage()).append('|');}
  }
  boolean a=false,b=false;
  try {value.concat(null);}catch(NullPointerException expected){a=true;}
  try {((String)null).substring(0,0);}catch(NullPointerException expected){b=true;}
  String empty=new String("");
  out.append(a).append(':').append(b).append(':').append(value.substring(1,1)=="")
   .append(':').append(empty.substring(0,0)==empty);
  System.out.print(out.toString());
 }
}`, `package main
import("fmt";"strings";j "github.com/NickyBoy89/java2go/stdjava")
func main(){
 value:=j.NewJavaStringUTF16([]uint16{'a','b'});var out strings.Builder
 for _,bounds:=range [][2]int32{{-1,1},{0,3},{2,1}} {
  func(){defer func(){if e:=recover();e!=nil {if !j.CaughtAs(e,"StringIndexOutOfBoundsException"){panic(e)};out.WriteString(j.GetMessage(e));out.WriteByte('|')}}();value.Substring(bounds[0],bounds[1]);out.WriteString("missing")}()
 }
 nullFailure:=func(f func())(caught bool){defer func(){caught=j.CaughtAs(recover(),"NullPointerException")}();f();return}
 a:=nullFailure(func(){value.Concat(nil)});b:=nullFailure(func(){var s *j.JavaString;s.Substring(0,0)})
 empty:=j.NewJavaStringUTF16(nil)
 fmt.Fprintf(&out,"%t:%t:%t:%t",a,b,value.Substring(1,1)==j.JavaStringLiteralUTF16(nil),empty.Substring(0,0)==empty)
 fmt.Print(out.String())
}`)
}
