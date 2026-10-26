package transpiler

import (
 "fmt"
 "testing"
)

// Every expected stream is obtained from the original JDK for the same source.
// Source declarations and value qualifiers remain in the same compiled control.
func TestCanonicalNormalizerNestedFormOwnerMatrix(t *testing.T) {
 for _, item := range []struct{name, source string}{
  {"ImportedNestedForm", `import java.text.Normalizer.Form;
public class ImportedNestedForm { public static String run(){String s="Ａe\u0301";return java.text.Normalizer.normalize(s,Form.NFC)+":"+java.text.Normalizer.normalize(s,Form.NFD)+":"+java.text.Normalizer.normalize(s,Form.NFKC)+":"+java.text.Normalizer.normalize(s,Form.NFKD);} }`},
  {"ImportedOuterForm", `import java.text.Normalizer;
public class ImportedOuterForm { public static String run(){String s="Ａe\u0301";return Normalizer.normalize(s,Normalizer.Form.NFC)+":"+Normalizer.normalize(s,Normalizer.Form.NFD)+":"+Normalizer.normalize(s,Normalizer.Form.NFKC)+":"+Normalizer.normalize(s,Normalizer.Form.NFKD);} }`},
  {"QualifiedFormsWithSourceOuter", `class Normalizer {static class Form {static int NFC=23;} static String normalize(String s,int n){return "source:"+s+":"+n;}}
public class QualifiedFormsWithSourceOuter { public static String run(){String s="Ａe\u0301";return Normalizer.normalize(s,Normalizer.Form.NFC)+":"+java.text.Normalizer.normalize(s,java.text.Normalizer.Form.NFC)+":"+java.text.Normalizer.normalize(s,java.text.Normalizer.Form.NFD)+":"+java.text.Normalizer.normalize(s,java.text.Normalizer.Form.NFKC)+":"+java.text.Normalizer.normalize(s,java.text.Normalizer.Form.NFKD);} }`},
  {"SourceSimpleForm", `import java.text.Normalizer;
class Form {static int NFC=19;}
public class SourceSimpleForm { public static String run(){return Form.NFC+":"+Normalizer.normalize("e\u0301",Normalizer.Form.NFC)+":"+java.text.Normalizer.normalize("Ａ",java.text.Normalizer.Form.NFKC);} }`},
  {"ValueQualifierForm", `import java.text.Normalizer;
class HolderForm {int NFC=31;}
class Holder {HolderForm Form=new HolderForm();}
public class ValueQualifierForm {static int calls;static Holder Normalizer=new Holder();static Holder next(){calls++;return Normalizer;}static String parameter(Holder Normalizer){return Normalizer.Form.NFC+":"+calls;}public static String run(){String value=parameter(next());return value+":"+Normalizer.Form.NFC+":"+calls+":"+java.text.Normalizer.normalize("e\u0301",java.text.Normalizer.Form.NFC);} }`},
 } {
  t.Run(item.name,func(t *testing.T){
   want:=campaignRuntimeJavaOracle(t,item.name,item.source)
   generated:=renderGoFileFromJava(t,item.source)
   runGoTestInTempModule(t,generated,fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestNestedFormOwners(t *testing.T){if got:=Run();!got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatal("nested owner or value qualifier stream differs from original JDK")}}
`,want))
  })
 }
}
