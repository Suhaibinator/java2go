package transpiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCampaignLocalIdentifierHygieneJVMParity(t *testing.T) {
	const source = `import java.util.function.Function; import java.util.function.Supplier;
class Parent {}
public class CampaignLocalNames {
 static String parameter(int string, int any){String value="p";Object broad=value;return string+":"+any+":"+(String)broad;}
 static String loops(){String result="";for(int string=0, any=1;string<2;string++){String item="x";result+=item+string+any;any++;}for(int bool:new int[]{3,4}){String item="y";result+=item+bool;}return result;}
 static String lambda(){Function<String,String> f=string->{String copy=string;return copy+"!";};Function<String,String> g=(String any)->{String copy=any;return copy+"?";};return f.apply("a")+g.apply("b");}
 static String caught(){try{throw new IllegalArgumentException("caught");}catch(IllegalArgumentException string){String result=string.getMessage();return result;}}
 static String repeated(){String result="";{int string=1;String text="a";result+=text+string;}{int string=2;String text="b";result+=text+string;}return result;}
 static String captures(){int string=6;Supplier<String> anon=new Supplier<String>(){public String get(){return "a"+string;}};class Local{String get(){return "b"+string;}}return anon.get()+new Local().get();}
 static String pattern(){Object object="ok";if(object instanceof String string){String copy=string;return copy;}return "bad";}
 static int varargs(int... string){return string[0]+string.length;}
 public static String run(){
  int string=7;int stringJava2goLocal=9;int bool=2;int any=4;int error=5;
  Parent parent=new Parent();Object broad=parent;Parent copy=(Parent)broad;
  String text=parameter(string,any);boolean same=copy==parent;
  return text+":"+stringJava2goLocal+":"+bool+":"+error+":"+same+":"+loops()+":"+lambda()+":"+caught()+":"+repeated()+":"+varargs(3,4)+":"+captures()+":"+pattern();
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignLocalNames", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestNames(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}

func TestCampaignLocalTypeShadowOriginalJVMParity(t *testing.T) {
	source, err := os.ReadFile("../campaign/reproducers/local-class-type-shadowing/LocalTypeShadow.java")
	if err != nil {
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
	root := t.TempDir()
	path := filepath.Join(root, "LocalTypeShadow.java")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(javac, "--release", "21", "-d", root, path).CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, output)
	}
	want, err := exec.Command(java, "-cp", root, "LocalTypeShadow").CombinedOutput()
	if err != nil {
		t.Fatalf("java: %v\n%s", err, want)
	}
	generated := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";"os";"io")
func TestOriginal(t *testing.T){old:=os.Stdout;r,w,err:=os.Pipe();if err!=nil{t.Fatal(err)};os.Stdout=w;defer func(){os.Stdout=old}();Main();if err:=w.Close();err!=nil{t.Fatal(err)};got,err:=io.ReadAll(r);if err!=nil{t.Fatal(err)};if err:=r.Close();err!=nil{t.Fatal(err)};if string(got)!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, string(want), string(want)))
}
