package transpiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"
)

func TestCampaignAssertionsJVMParity(t *testing.T) {
	const source = `class AssertionDetail{public String toString(){CampaignAssertions.conversions++;return "detail";}}
public class CampaignAssertions{
 static int conditions;static int messages;static int conversions;
 static boolean condition(boolean value){conditions++;return value;}
 static Object detail(){messages++;return new AssertionDetail();}
 public static String run(){String result="";
  assert condition(true):detail();
  try{assert condition(false):detail();result+="disabled";}catch(AssertionError error){result+=error.getMessage();}
  try{assert false;result+=":off";}catch(AssertionError error){String message=error.getMessage();result+=":"+(message==null);}
  try{assert false:'X';result+=":off";}catch(AssertionError error){result+=":"+error.getMessage();}
  RuntimeException cause=new RuntimeException("cause");
  try{assert false:cause;result+=":off";}catch(AssertionError error){result+=":"+(error.getCause()==cause);}
  try{assert false:null;result+=":off";}catch(AssertionError error){result+=":"+error.getMessage();}
  return result+":"+conditions+":"+messages+":"+conversions;
 }
}`
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%v", enabled), func(t *testing.T) {
			t.Setenv("JAVA2GO_ASSERTIONS", fmt.Sprint(enabled))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			root := t.TempDir()
			file := filepath.Join(root, "CampaignAssertions.java")
			if err := os.WriteFile(file, []byte(source+`class AssertionDriver{public static void main(String[]args){System.out.print(CampaignAssertions.run());}}`), 0600); err != nil {
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
			if out, err := exec.CommandContext(ctx, javac, "--release", "21", "-d", root, file).CombinedOutput(); err != nil {
				t.Fatalf("javac: %v\n%s", err, out)
			}
			args := []string{"-cp", root}
			if enabled {
				args = append(args, "-ea")
			}
			args = append(args, "AssertionDriver")
			want, err := exec.CommandContext(ctx, java, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("JVM: %v\n%s", err, want)
			}
			t.Logf("JVM assertions enabled=%v: %s", enabled, want)
			generated := renderGoFileFromJava(t, source)
			runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestAssertions(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != Go UTF16 %%#v", %q, units) }
}`, utf16.Encode([]rune(string(want))), string(want)))
		})
	}
}
