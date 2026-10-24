package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCampaignCanonicalNPEConstructorLowering(t *testing.T) {
	source := `public class NPELowering {static NullPointerException empty(){return new NullPointerException();}static NullPointerException detail(String text){return new NullPointerException(text);}static NullPointerException absent(){return new NullPointerException(null);}static <T extends String> NullPointerException bounded(T text){return new NullPointerException(text);}}`
	generated := renderGoFileFromJava(t, source)
	if !strings.Contains(generated, "stdjava.NewJavaNullPointerExceptionMessage(") || strings.Contains(generated, "stdjava.NewNullPointerException(") {
		t.Fatalf("NPE constructor must select canonical String ABI:\n%s", generated)
	}
	enum := renderGoFileFromJava(t, `public enum NPEChoice {ONE}`)
	for _, required := range []string{"stdjava.NewJavaNullPointerExceptionMessage(", "stdjava.NewJavaIllegalArgumentExceptionMessage("} {
		if !strings.Contains(enum, required) {
			t.Fatalf("missing canonical synthetic enum constructor %q in:\n%s", required, enum)
		}
	}
	for _, legacy := range []string{"stdjava.NewNullPointerException(", "stdjava.NewIllegalArgumentException("} {
		if strings.Contains(enum, legacy) {
			t.Fatalf("native synthetic enum constructor %q in:\n%s", legacy, enum)
		}
	}
}

func TestCampaignCanonicalNPERuntimeJDK21(t *testing.T) {
	campaignStringCoreOperationOracle(t, "CanonicalNPERuntime", `public class CanonicalNPERuntime {
 static String units(String text){if(text==null)return "null";String out="";for(int i=0;i<text.length();i++)out=out+(int)text.charAt(i)+",";return out;}
 static void record(String message){NullPointerException value=new NullPointerException(message);RuntimeException cause=new RuntimeException("cause");boolean before=value.getCause()==null;boolean same=value.initCause(cause)==value;boolean second=false;try{value.initCause(null);}catch(IllegalStateException blocked){second=true;}System.out.println((value.getMessage()==message)+":"+units(value.getMessage())+":"+units(value.toString())+":"+before+":"+same+":"+(value.getCause()==cause)+":"+second);}
 public static void main(String[] args){String[] messages={null,"","Name is null",new String(new char[]{'h',(char)0xd800,'l',(char)0xdc00,0}),new String(new char[]{(char)0xd800,(char)0xdc00})};for(String message:messages)record(message);}
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func units(text *j.JavaString)string{if text==nil{return "null"};out:="";for _,u:=range text.UTF16Copy(){out+=fmt.Sprintf("%d,",u)};return out}
func record(message *j.JavaString){e:=j.NewExecution();value:=j.NewJavaNullPointerExceptionMessage(message);cause:=j.NewJavaRuntimeExceptionMessage(j.JavaStringFromHostUTF8("cause"));before:=j.GetCause(value)==nil;same:=j.JavaReferenceEqual(j.ThrowableInitCauseExecution(e,value,cause),value);second:=func()(blocked bool){defer func(){blocked=j.CaughtAs(recover(),"IllegalStateException")}();j.ThrowableInitCauseExecution(e,value,nil);return}();fmt.Printf("%t:%s:%s:%t:%t:%t:%t\n",j.JavaThrowableMessageDefault(value)==message,units(j.JavaThrowableMessageDefault(value)),units(j.JavaThrowableToStringExecution(e,value)),before,same,j.JavaReferenceEqual(j.GetCause(value),cause),second)}
func main(){messages:=[]*j.JavaString{nil,j.JavaStringLiteralUTF16(nil),j.JavaStringFromHostUTF8("Name is null"),j.NewJavaStringUTF16([]uint16{'h',0xd800,'l',0xdc00,0}),j.NewJavaStringUTF16([]uint16{0xd800,0xdc00})};for _,message:=range messages{record(message)};if j.NewNullPointerException("native").Message()!="native"{panic("native constructor changed")}}
`)
}

func TestCampaignCanonicalNPEEmissionJDK21(t *testing.T) {
	const source = `package canonicalnpe;
class NPESub extends NullPointerException {NPESub(String message){super(message);}}
class NPEDefaultSub extends NullPointerException {}
enum NPEChoice {ONE}
public class CanonicalNPEEmission {
 static String units(String text){if(text==null)return "null";String out="";for(int i=0;i<text.length();i++)out=out+(int)text.charAt(i)+",";return out;}
 static <T extends String> NullPointerException bounded(T message){return new NullPointerException(message);}
 static String record(NullPointerException value,String message,Throwable cause){boolean before=value.getCause()==null;boolean same=value.initCause(cause)==value;boolean second=false;try{value.initCause(null);}catch(IllegalStateException blocked){second=true;}return (value.getMessage()==message)+":"+units(value.getMessage())+":"+units(value.toString())+":"+before+":"+same+":"+(value.getCause()==cause)+":"+second;}
 static String choice(String name,Throwable cause){try{return NPEChoice.valueOf(name).name();}catch(RuntimeException failure){boolean before=failure.getCause()==null;boolean same=failure.initCause(cause)==failure;return failure.getClass().getSimpleName()+":"+units(failure.getMessage())+":"+before+":"+same+":"+(failure.getCause()==cause);}}
 public static String run(){String message=new String(new char[]{'h',(char)0xd800,'l',(char)0xdc00,0});Throwable cause=new RuntimeException("cause");String out=record(new NullPointerException(),null,cause)+"\n"+record(new NullPointerException((String)null),null,cause)+"\n"+record(new NullPointerException(null),null,cause)+"\n"+record(new NullPointerException(""),"",cause)+"\n"+record(new NullPointerException(message),message,cause)+"\n"+record(new NPESub(message),message,cause)+"\n"+record(new NPEDefaultSub(),null,cause)+"\n"+record(bounded(message),message,cause);return out+"\n"+choice(null,cause)+"\n"+choice(message,cause)+"\n"+choice("missing",cause)+"\n"+choice("ONE",cause)+"\n";}
 public static void main(String[] args){System.out.print(run());}
}`
	want := canonicalNPEProjectOracle(t, source)
	t.Logf("JVM canonical NPE emission oracle: %q", want)
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>npe-emission</artifactId><version>1</version></project>`,
		"src/main/java/canonicalnpe/CanonicalNPEEmission.java": source,
	}, "canonicalnpe.CanonicalNPEEmission", want)
}

// The strict Maven project uses a named Java package. Run its exact source with
// the matching qualified entry point before building either generated program.
func canonicalNPEProjectOracle(t *testing.T, source string) string {
	t.Helper()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "CanonicalNPEEmission.java")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	campaignStringRequireSuccess(t, "NPE project javac", campaignStringBoundedRun(t, "NPE project javac", dir, javac, 60*time.Second, "--release", "21", "-encoding", "UTF-8", "-d", dir, path))
	observed := campaignStringBoundedRun(t, "NPE project JVM", dir, java, 60*time.Second, "-cp", dir, "canonicalnpe.CanonicalNPEEmission")
	campaignStringRequireSuccess(t, "NPE project JVM", observed)
	if len(observed.stderr) != 0 {
		t.Fatalf("NPE project JVM stderr: %q", observed.stderr)
	}
	return string(observed.stdout)
}
