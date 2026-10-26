package transpiler

import (
 "fmt"
 "os"
 "testing"
)

// Keep the complete independently compiled Java witness. Each generated
// program runs in a fresh Go test process against its actual JDK21 stream.
func TestQualifiedSourceReceiverEnclosingApplicationParity(t *testing.T) {
 for _, mode := range []string{"instance", "static", "priorlocal"} {
  t.Run(mode,func(t *testing.T) {
   source,err:=os.ReadFile("testdata/enclosing_qualifier_witness/Main.java")
   if err!=nil {t.Fatal(err)}
   want,err:=os.ReadFile("testdata/enclosing_qualifier_witness/"+mode+".stdout")
   if err!=nil {t.Fatal(err)}
   generated:=renderGoFileFromJava(t,string(source))
   runGoTestInTempModule(t,generated,fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestActualEnclosingProgram(t *testing.T){got:=Run(j.JavaStringFromHostUTF8(%q));if !got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatal("actual enclosing program differs from original JDK21")}}
`,mode,string(want)))
  })
 }
}
