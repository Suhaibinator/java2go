package transpiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"
)

func assertionStatementReferenceModesJDK21(t *testing.T, name, source, disabled, enabled string) {
	t.Helper()
	for _, mode := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%v", mode), func(t *testing.T) {
			t.Setenv("JAVA2GO_ASSERTIONS", fmt.Sprint(mode))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			root := t.TempDir()
			path := filepath.Join(root, name+".java")
			if err := os.WriteFile(path, []byte(source+"\nclass AssertionReferenceDriver{public static void main(String[] args){System.out.print("+name+".run());}}"), 0600); err != nil {
				t.Fatal(err)
			}
			javac, err := campaignCompilerJavaTool("javac")
			if err != nil {
				t.Fatal(err)
			}
			java, err := campaignCompilerJavaTool("java")
			if err != nil {
				t.Fatal(err)
			}
			compile := exec.CommandContext(ctx, javac, "--release", "21", "-encoding", "UTF-8", "-d", root, path)
			if output, err := compile.CombinedOutput(); err != nil {
				t.Fatalf("JDK compile: %v\n%s", err, output)
			}
			args := []string{"-cp", root}
			if mode {
				args = append(args, "-ea")
			}
			args = append(args, "AssertionReferenceDriver")
			cmd := exec.CommandContext(ctx, java, args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("JDK run: %v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
			}
			want := disabled
			if mode {
				want = enabled
			}
			if ctx.Err() != nil || stdout.String() != want || stderr.Len() != 0 {
				t.Fatalf("JDK oracle: timeout=%v stdout=%q stderr=%q want=%q", ctx.Err(), stdout.String(), stderr.String(), want)
			}
			t.Logf("JDK assertion contract mode=%v argv=%v stdout=%q stderr=%q", mode, cmd.Args, stdout.String(), stderr.String())
			strictRoutingState(t)
			generated := renderGoFileFromJava(t, source)
			runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("slices";"testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestAssertionReference(t *testing.T){var got *j.JavaString=Run();if got==nil{t.Fatal("Run returned null")};if units:=got.UTF16Copy();!slices.Equal(units,%#v){t.Fatalf("JDK UTF16 != Go UTF16: %%#v",units)}}`, utf16.Encode([]rune(want))))
		})
	}
}

func TestCampaignAssertionStatementReferenceDetailsJDK21(t *testing.T) {
	const source = `public class AssertionStatementReferenceDetails {
 static int calls;static int conditions;static int details;
 static RuntimeException replacement=new RuntimeException("replacement");
 static boolean condition(){conditions++;return false;}static Object touch(){details++;return null;}
 static class ParentDetail {String text;ParentDetail(String value){text=value;}public String toString(){calls++;return text;}}
 static class ChildDetail extends ParentDetail {ChildDetail(String value){super(value);}}
 static class NullDetail {public String toString(){calls++;return null;}}
 static class Cause extends RuntimeException {String text;Cause(String value){text=value;}public String toString(){calls++;return text;}}
 static AssertionError failure(Object value){try{assert false:value;throw replacement;}catch(AssertionError error){return error;}}
 static AssertionError empty(){try{assert false;throw replacement;}catch(AssertionError error){return error;}}
 static boolean available(AssertionError error){try{return error.initCause(replacement)==error&&error.getCause()==replacement;}catch(IllegalStateException expected){return false;}}
 static int character(char value){try{assert false:value;return -1;}catch(AssertionError error){return error.getMessage().charAt(0);}}
 static String primitive(){String result="";boolean truth=true;byte small=-2;short medium=3;int integer=4;long wide=5L;float single=-0.0f;double decimal=1.25;
  try{assert false:truth;}catch(AssertionError e){result+=e.getMessage();}
  try{assert false:small;}catch(AssertionError e){result+=":"+e.getMessage();}
  try{assert false:medium;}catch(AssertionError e){result+=":"+e.getMessage();}
  try{assert false:integer;}catch(AssertionError e){result+=":"+e.getMessage();}
  try{assert false:wide;}catch(AssertionError e){result+=":"+e.getMessage();}
  try{assert false:single;}catch(AssertionError e){result+=":"+e.getMessage();}
  try{assert false:decimal;}catch(AssertionError e){result+=":"+e.getMessage();}return result;
 }
 public static String run(){boolean enabled=false;assert enabled=true;if(!enabled){assert condition():touch();return "disabled="+conditions+":"+details;}
  String text=new String(new char[]{'r',0,(char)0xd800,'x',(char)0xdc00});
  AssertionError direct=failure(text);
  String result="detail="+(direct.getMessage()==text)+":"+direct.getMessage().length()+":"+(int)direct.getMessage().charAt(1)+":"+(int)direct.getMessage().charAt(2)+":"+(int)direct.getMessage().charAt(4)+":"+available(direct);
  AssertionError inherited=failure(new ChildDetail(text));Cause cause=new Cause(text);AssertionError caused=failure(cause);AssertionError nullResult=failure(new NullDetail());
  result+="|source="+(inherited.getMessage()==text)+":"+available(inherited)+":"+(caused.getCause()==cause)+":"+(caused.getMessage()==text)+":"+(!available(caused))+":"+calls;
  AssertionError nullObject=failure(null);AssertionError nullString=failure((String)null);AssertionError noDetail=empty();
  result+="|null="+(nullObject.getMessage()=="null")+":"+(nullString.getMessage()=="null")+":"+(nullResult.getMessage()==null)+":"+(noDetail.getMessage()==null)+":"+available(noDetail)+":"+available(nullObject);
  return result+"|char="+character((char)0)+":"+character((char)0xd800)+":"+character((char)0xdc00)+"|primitive="+primitive();
 }
}`
	assertionStatementReferenceModesJDK21(t, "AssertionStatementReferenceDetails", source, "disabled=0:0", "detail=true:5:0:55296:56320:true|source=true:true:true:true:true:3|null=true:true:true:true:true:true|char=0:55296:56320|primitive=true:-2:3:4:5:-0.0:1.25")
}

func TestCampaignAssertionStatementCallerAndCleanupJDK21(t *testing.T) {
	const source = `public class AssertionStatementCallerAndCleanup {
 static ThreadLocal<String> local=new ThreadLocal<String>();static Thread caller;static String trace="";
 static String text=new String(new char[]{'r',0,(char)0xd800});static RuntimeException marker=new RuntimeException("marker");
 static int conditions;static int details;static int renders;
 static boolean condition(boolean value){conditions++;trace+="condition,";return value;}
 static Object detail(Object value){details++;trace+="detail,";return value;}
 static class Detail {boolean held;boolean same;boolean contextual;boolean abrupt;public String toString(){renders++;held=Thread.holdsLock(this);same=Thread.currentThread()==caller;contextual=local.get()==text;trace+="render,";if(abrupt)throw marker;return text;}}
 public static String run(){boolean enabled=false;assert enabled=true;caller=Thread.currentThread();Detail value=new Detail();
  if(!enabled){assert condition(false):detail(value);return "disabled="+conditions+":"+details+":"+renders;}
  AssertionError built=null;
  synchronized(value){local.set(text);trace+="enter,";try{assert condition(true):detail(value);assert condition(false):detail(value);}catch(AssertionError error){built=error;trace+="catch,";}finally{local.remove();trace+="finally,";}}
  synchronized(value){trace+="reacquire,";}
  String result="caller="+(built.getMessage()==text)+":"+value.held+":"+value.same+":"+value.contextual+":"+(local.get()==null)+":"+Thread.holdsLock(value)+":"+conditions+":"+details+":"+renders+"|"+trace;
  trace="";Detail broken=new Detail();broken.abrupt=true;boolean caught=false;
  try{synchronized(broken){local.set(text);trace+="enter,";try{assert condition(false):detail(broken);}finally{local.remove();trace+="finally,";}}}catch(RuntimeException failure){caught=failure==marker;trace+="catch-marker,";}
  synchronized(broken){trace+="reacquire,";}
  return result+"|abrupt="+caught+":"+broken.held+":"+broken.same+":"+broken.contextual+":"+(local.get()==null)+":"+Thread.holdsLock(broken)+":"+conditions+":"+details+":"+renders+"|"+trace;
 }
}`
	assertionStatementReferenceModesJDK21(t, "AssertionStatementCallerAndCleanup", source, "disabled=0:0:0", "caller=true:true:true:true:true:false:2:1:1|enter,condition,condition,detail,render,catch,finally,reacquire,|abrupt=true:true:true:true:true:false:3:2:2|enter,condition,detail,render,finally,catch-marker,reacquire,")
}
