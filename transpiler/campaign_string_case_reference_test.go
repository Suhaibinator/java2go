package transpiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func stringCaseReferenceOracle(t *testing.T, class, source string) {
	t.Helper()
	want := stringCaseReferenceJavaOracle(t, class, source)
	t.Logf("JDK21 canonical case oracle %s=%q", class, want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("slices";"testing";"unicode/utf16";j "github.com/NickyBoy89/java2go/stdjava")
func TestStringCaseOracle(t *testing.T){var got *j.JavaString=Run();if got==nil{t.Fatal("Run returned null")};want:=utf16.Encode([]rune(%q));if units:=got.UTF16Copy();!slices.Equal(units,want){t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x",want,units)}}`, want))
}

const stringCaseEnglishSource = `import java.util.Locale;
public class StringCaseEnglish {
 static String units(String value){StringBuilder out=new StringBuilder();for(int i=0;i<value.length();i++)out.append((int)value.charAt(i)).append(',');return out.toString();}
 public static String run(){
  String[] values={"","already lower","ALREADY UPPER","straße ﬃ","ΟΣ","ΟΣΑ","ΟΣ'","ΌΣ","İ","iIıİ",new String(new char[]{'a',(char)0xd800,'B',(char)0xdc00,'c'}),new String(new char[]{(char)0xd801,(char)0xdc00,(char)0xd801,(char)0xdc28}),new String(new char[]{(char)0xfffd,'a'}),"Ο\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301Σ"};
  StringBuilder out=new StringBuilder();
  for(String value:values){String upper=value.toUpperCase(Locale.ENGLISH),lower=value.toLowerCase(Locale.ENGLISH);out.append(units(upper)).append(':').append(upper==value).append('/').append(units(lower)).append(':').append(lower==value).append('|').append(units(value)).append(';');}
  return out.toString();
 }
}`

const stringCaseDefaultSource = `import java.util.Locale;
public class StringCaseDefault {
 static String units(String value){StringBuilder out=new StringBuilder();for(int i=0;i<value.length();i++)out.append((int)value.charAt(i)).append(',');return out.toString();}
 public static String run(){Locale old=Locale.getDefault();try{
  String value="iIıİ";Locale.setDefault(Locale.forLanguageTag("tr"));String first=value.toUpperCase(),second=value.toLowerCase();
  Locale.setDefault(Locale.ENGLISH);String third=value.toUpperCase(),fourth=value.toLowerCase();
  Locale.setDefault(Locale.forLanguageTag("lt"));String fifth="I\u0301 J\u0300".toLowerCase(),sixth="i\u0307\u0301".toUpperCase();
  String seventh="άι".toUpperCase(Locale.forLanguageTag("el"));
  return units(first)+"/"+units(second)+"/"+units(third)+"/"+units(fourth)+"/"+units(fifth)+"/"+units(sixth)+"/"+units(seventh);
 }finally{Locale.setDefault(old);}}
}`

const stringCaseContractsSource = `import java.util.Locale;
public class StringCaseContracts {
 static String trace="";static String stored;
 static String receiver(int mode){trace+="R";return mode<2?null:new String("source");}
 static Locale locale(int mode){trace+="L";if(mode==1)throw new IllegalStateException();return mode==3?null:Locale.ENGLISH;}
 static Locale mutate(){trace+="M";stored=null;return Locale.ENGLISH;}
 public static String run(){StringBuilder out=new StringBuilder();for(int mode=0;mode<4;mode++){
  trace="";try{receiver(mode).toUpperCase(locale(mode));trace+="V";}catch(IllegalStateException e){trace+="E";}catch(NullPointerException e){trace+="N";}
  out.append(trace).append('|');
 }
 trace="";stored="lower";String result=stored.toUpperCase(mutate());out.append(trace).append(':').append(result).append(':').append(stored==null);
 String same=new String("123 ");out.append('|').append(same.toLowerCase(Locale.ENGLISH)==same).append(':').append(same.toUpperCase(Locale.ENGLISH)==same);
 return out.toString();}
}`

const stringCaseShadowStringSource = `class String {
 java.lang.String toUpperCase(){return "source-upper";}
 java.lang.String toLowerCase(java.util.Locale locale){return "source-lower";}
}
public class StringCaseShadowString {public static java.lang.String run(){String value=new String();return value.toUpperCase()+"/"+value.toLowerCase(java.util.Locale.ENGLISH)+"/"+new java.lang.String("builtin").toUpperCase(java.util.Locale.ENGLISH);}}`

const stringCaseShadowLocaleSource = `class Locale {static java.lang.String marker(){return "source-locale";}}
public class StringCaseShadowLocale {public static String run(){return "builtin".toLowerCase(java.util.Locale.ENGLISH)+"/"+Locale.marker();}}`

func TestCampaignStringCaseEnglishJVMParity(t *testing.T) {
	stringCaseReferenceOracle(t, "StringCaseEnglish", stringCaseEnglishSource)
}
func TestCampaignStringCaseDefaultLocaleJVMParity(t *testing.T) {
	stringCaseReferenceOracle(t, "StringCaseDefault", stringCaseDefaultSource)
}
func TestCampaignStringCaseContractsJVMParity(t *testing.T) {
	stringCaseReferenceOracle(t, "StringCaseContracts", stringCaseContractsSource)
}
func TestCampaignStringCaseSourceStringShadowJVMParity(t *testing.T) {
	stringCaseReferenceOracle(t, "StringCaseShadowString", stringCaseShadowStringSource)
}
func TestCampaignStringCaseSourceLocaleShadowJVMParity(t *testing.T) {
	stringCaseReferenceOracle(t, "StringCaseShadowLocale", stringCaseShadowLocaleSource)
}
func TestCampaignStringCaseCanonicalDispatchAST(t *testing.T) {
	generated := renderGoFileFromJava(t, stringCaseShadowStringSource)
	if got := strings.Count(generated, "stdjava.JavaStringToUpperCase("); got != 1 {
		t.Fatalf("expected one qualified builtin case call, source String methods must remain source calls: count=%d\n%s", got, generated)
	}
	if strings.Contains(generated, "stdjava.JavaStringToLowerCase(") {
		t.Fatalf("source String.toLowerCase was rewritten to builtin case service:\n%s", generated)
	}
}

func stringCaseReferenceJavaOracle(t *testing.T, name, source string) string {
	t.Helper()
	home := os.Getenv("JAVA_HOME")
	javac, java := "javac", "java"
	if home != "" {
		javac, java = filepath.Join(home, "bin", "javac"), filepath.Join(home, "bin", "java")
	}
	for _, tool := range []*string{&javac, &java} {
		resolved, err := exec.LookPath(*tool)
		if err != nil {
			t.Fatalf("campaign requires JDK 21: cannot find %s (set JAVA_HOME): %v", *tool, err)
		}
		*tool = resolved
	}
	dir := t.TempDir()
	driver := `class CampaignRuntimeOracle { public static void main(java.lang.String[] args) throws Exception { System.out.print(` + name + `.run()); } }`
	path := filepath.Join(dir, name+".java")
	if err := os.WriteFile(path, []byte(source+"\n"+driver), 0600); err != nil {
		t.Fatal(err)
	}
	compileContext, cancelCompile := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelCompile()
	if out, err := exec.CommandContext(compileContext, javac, "--release", "21", "-encoding", "UTF-8", "-d", dir, path).CombinedOutput(); err != nil {
		t.Fatalf("JDK 21 oracle compilation: %v\n%s", err, out)
	}
	runContext, cancelRun := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRun()
	out, err := exec.CommandContext(runContext, java, "-cp", dir, "CampaignRuntimeOracle").CombinedOutput()
	if err != nil {
		t.Fatalf("JDK oracle execution: %v\n%s", err, out)
	}
	return string(out)
}
