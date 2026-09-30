package transpiler

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This is a handwritten runtime output driver, not compiler acceptance. Both
// programs receive actual process arguments and the oracle pins stdout UTF-8.
func campaignStringHostOracle(t *testing.T, class, javaSource, goDriver string, arguments ...string) {
	t.Helper()
	java, javac := "/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/java", "/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/bin/javac"
	temporary := t.TempDir()
	javaPath := filepath.Join(temporary, class+".java")
	if err := os.WriteFile(javaPath, []byte(javaSource), 0600); err != nil {
		t.Fatal(err)
	}
	campaignStringRequireSuccess(t, "host javac", campaignStringBoundedRun(t, "host javac", temporary, javac, 300*time.Second, "--release", "21", "-encoding", "UTF-8", "-d", temporary, javaPath))
	javaArguments := append([]string{"-Dfile.encoding=UTF-8", "-Dstdout.encoding=UTF-8", "-cp", temporary, class}, arguments...)
	want := campaignStringBoundedRun(t, "host JVM", temporary, java, 60*time.Second, javaArguments...)
	campaignStringRequireSuccess(t, "host JVM", want)
	if len(want.stderr) != 0 {
		t.Fatalf("unexpected JVM stderr: %q", want.stderr)
	}
	t.Logf("JVM host oracle=%q", want.stdout)
	module := "module hostprobe\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	for name, content := range map[string]string{"go.mod": module, "main.go": goDriver} {
		if err := os.WriteFile(filepath.Join(temporary, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	executable := filepath.Join(temporary, "host-boundary")
	campaignStringRequireSuccess(t, "host runtime driver build", campaignStringBoundedRun(t, "host runtime driver build", temporary, "go", 300*time.Second, "build", "-race", "-mod=mod", "-o", executable, "."))
	got := campaignStringBoundedRun(t, "host runtime driver", temporary, executable, 60*time.Second, arguments...)
	if got.exit != want.exit || !bytes.Equal(got.stdout, want.stdout) || !bytes.Equal(got.stderr, want.stderr) {
		t.Fatalf("JVM exit%d stdout%q stderr%q; runtime exit%d stdout%q stderr%q", want.exit, want.stdout, want.stderr, got.exit, got.stdout, got.stderr)
	}
}

func TestCampaignStringHostActualArguments(t *testing.T) {
	campaignStringHostOracle(t, "StringHostArguments", `import java.nio.charset.*;
public class StringHostArguments {
 public static void main(String[] args) {
  if(!System.out.charset().equals(StandardCharsets.UTF_8)||!Charset.defaultCharset().equals(StandardCharsets.UTF_8))throw new AssertionError("UTF-8 required");
  for(String value:args){System.out.print((value==null)+":"+value.length()+":");for(int i=0;i<value.length();i++)System.out.print((int)value.charAt(i)+",");System.out.print("/");System.out.println(value);}
 }
}`, `package main
import("fmt";"os";j "github.com/NickyBoy89/java2go/stdjava")
func main(){for _,argument:=range os.Args[1:]{value:=j.JavaStringFromHostUTF8(argument);fmt.Printf("%t:%d:",value==nil,value.Length());for _,unit:=range value.UTF16Copy(){fmt.Printf("%d,",unit)};fmt.Print("/");j.JavaPrintlnStrings(value)}}`, "", "null", "ASCII", "λ😀", "line\nbreak")
}

func TestCampaignStringHostOutputUTF16(t *testing.T) {
	campaignStringHostOracle(t, "StringHostOutput", `import java.nio.charset.*;
public class StringHostOutput {
 static String value(char... units){return new String(units);}
 public static void main(String[] args){
  if(!System.out.charset().equals(StandardCharsets.UTF_8))throw new AssertionError("UTF-8 required");
  String[] values={null,"","null",value('a',(char)0,'b'),value((char)0xd83d,(char)0xde00),value((char)0xd800,'x',(char)0xdc00),value((char)0xd800,(char)0xd801,(char)0xdc00,(char)0xdc01),value((char)0xfffd)};
  for(String v:values){System.out.print("[");System.out.print(v);System.out.println("]");if(v!=null){for(int i=0;i<v.length();i++)System.out.print((int)v.charAt(i)+",");}System.out.println();}
  System.out.print("");System.out.println();System.out.println("");
  System.out.print("left");System.out.print((String)null);System.out.print("");System.out.print("right");System.out.println();
  System.out.println("left"+(String)null+""+"right");
  System.out.print(value((char)0xd83d));System.out.print(value((char)0xde00));System.out.println();
  System.out.println(value((char)0xd83d)+value((char)0xde00));
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func v(units ...uint16)*j.JavaString{return j.NewJavaStringUTF16(units)}
func main(){values:=[]*j.JavaString{nil,v(),v('n','u','l','l'),v('a',0,'b'),v(0xd83d,0xde00),v(0xd800,'x',0xdc00),v(0xd800,0xd801,0xdc00,0xdc01),v(0xfffd)}
 for _,value:=range values{j.JavaPrintStrings(v('['));j.JavaPrintStrings(value);j.JavaPrintlnStrings(v(']'));if value!=nil{for _,unit:=range value.UTF16Copy(){fmt.Printf("%d,",unit)}};j.JavaPrintlnStrings()}
 j.JavaPrintStrings();j.JavaPrintlnStrings();j.JavaPrintlnStrings(v())
 j.JavaPrintStrings(v('l','e','f','t'),nil,v(),v('r','i','g','h','t'));j.JavaPrintlnStrings()
 j.JavaPrintlnStrings(v('l','e','f','t'),nil,v(),v('r','i','g','h','t'))
 j.JavaPrintStrings(v(0xd83d));j.JavaPrintStrings(v(0xde00));j.JavaPrintlnStrings()
 j.JavaPrintlnStrings(v(0xd83d),v(0xde00))
}`)
}

func TestCampaignStringHostNativeWriterCompatibility(t *testing.T) {
	campaignStringHostOracle(t, "StringHostNativeWriters", `import java.io.*;import java.nio.charset.*;
public class StringHostNativeWriters {
 public static void main(String[] args)throws Exception{
  StringWriter text=new StringWriter();PrintWriter writer=new PrintWriter(text);writer.print("old");writer.println("native");writer.println("");writer.print("λ😀");writer.flush();System.out.print(text.toString());
  ByteArrayOutputStream bytes=new ByteArrayOutputStream();PrintStream stream=new PrintStream(bytes,false,StandardCharsets.UTF_8);stream.print("old");stream.println("native");stream.println("");stream.print("λ😀");stream.flush();System.out.write(bytes.toByteArray());
 }
}`, `package main
import("bytes";"fmt";j "github.com/NickyBoy89/java2go/stdjava")
func main(){text:=j.NewStringWriter();writer:=j.NewPrintWriter(text);writer.Print("old");writer.Println("native");writer.Println("");writer.Print("λ😀");writer.Flush();fmt.Print(text.String());var data bytes.Buffer;stream:=j.NewPrintStream(&data);stream.Print("old");stream.Println("native");stream.Println("");stream.Print("λ😀");stream.Flush();fmt.Print(data.String())}`)
}
