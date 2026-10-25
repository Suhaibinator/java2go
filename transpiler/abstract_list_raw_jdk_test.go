package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbstractListActualGsonRawAliasJDK21(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "gson_generic_parent_origin", "dependencies", "src", "com", "google", "gson", "internal", "NonNullElementWrapperList.java"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-raw</artifactId><version>1</version></project>`,
		"src/main/java/com/google/gson/internal/NonNullElementWrapperList.java": string(source),
		"src/main/java/probe/RawMain.java": `package probe;import java.util.*;import com.google.gson.internal.NonNullElementWrapperList;
  @SuppressWarnings({"rawtypes","unchecked"}) public class RawMain {public static void main(String[]args){
   ArrayList<String> d=new ArrayList<>();d.add("a");NonNullElementWrapperList<String> typed=new NonNullElementWrapperList<>(d);List raw=typed;
   raw.set(0,Integer.valueOf(7));System.out.println("raw-write:"+raw.get(0)+":"+(raw.get(0)==Integer.valueOf(7)));
   try{String text=typed.get(0);System.out.println("bad:"+text);}catch(ClassCastException e){System.out.println("typed-read:"+e.getClass().getSimpleName());}
   typed.set(0,"a");System.out.println("restored:"+raw.get(0)+":"+(raw==typed));raw.add(Integer.valueOf(9));System.out.println("raw-add:"+raw.get(1)+":"+raw.size());
   try{raw.set(99,null);System.out.println("bad-null");}catch(NullPointerException e){System.out.println("raw-null:"+e.getMessage()+":"+raw.size());}
  }}
 `}, "probe.RawMain", "raw-write:7:true\ntyped-read:ClassCastException\nrestored:a:true\nraw-add:9:2\nraw-null:Element must be non-null:2\n")
}
