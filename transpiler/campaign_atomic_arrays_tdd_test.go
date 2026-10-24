package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCampaignAtomicArraysStrictJVM(t *testing.T) {
	root := filepath.Join("testdata", "campaign", "atomic_arrays_v1")
	files := map[string]string{}
	for _, name := range []string{"pom.xml", "src/main/java/probe/AtomicArraysProbe.java"} {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(contents)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "probe.AtomicArraysProbe", []campaignCompilerProjectObservation{
		{name: "repeat-1", stdout: string(expected)}, {name: "repeat-2", stdout: string(expected)}, {name: "repeat-3", stdout: string(expected)},
	})
}

func TestCampaignAtomicArraysBoxedInvocationJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import java.util.concurrent.atomic.*;public class Main {
  static String trace="";
  static int[] copyI(){trace+="i";return new int[]{7,11};}
  static long[] copyL(){trace+="l";return new long[]{9007199254740993L,13L};}
  public static void main(String[] args){
   Integer size=2;short idx=1;Integer value=17;Long wide=9007199254740997L;
   AtomicIntegerArray i=new AtomicIntegerArray(size);AtomicLongArray l=new AtomicLongArray(size);
   i.set(idx,value);l.set(idx,wide);
   System.out.println(i.get(idx)+":"+l.get(idx)+":"+l.getAndAdd(idx,value)+":"+l.get(idx));
   AtomicIntegerArray ci=new AtomicIntegerArray(copyI());AtomicLongArray cl=new AtomicLongArray(copyL());
   System.out.println(trace+":"+ci.get(0)+":"+cl.get(0));
   Integer absent=null;
   try{i.set(absent,value);}catch(NullPointerException expected){System.out.println("null-index");}
   try{new AtomicLongArray(absent);}catch(NullPointerException expected){System.out.println("null-length");}
  }
 }`,
	}, "probe.Main", "17:9007199254740997:9007199254740997:9007199254741014\nil:7:9007199254740993\nnull-index\nnull-length\n")
}

func TestCampaignAtomicArraysSourceForeignBinderGuardsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/foreign/AtomicLongArray.java": `package foreign;public class AtomicLongArray{public AtomicLongArray(int length){}public long get(int index){return 73;}}`,
		"src/main/java/probe/Main.java": `package probe;import foreign.AtomicLongArray;public class Main {
  static class AtomicIntegerArray{AtomicIntegerArray(int length){}int get(int index){return 71;}}
  static class Holder<AtomicIntegerArray>{AtomicIntegerArray value;Holder(AtomicIntegerArray value){this.value=value;}AtomicIntegerArray get(){return value;}}
  public static void main(String[] args){
   AtomicIntegerArray local=new AtomicIntegerArray(1);AtomicLongArray foreign=new AtomicLongArray(1);
   java.util.concurrent.atomic.AtomicIntegerArray nativeI=new java.util.concurrent.atomic.AtomicIntegerArray(1);
   java.util.concurrent.atomic.AtomicLongArray nativeL=new java.util.concurrent.atomic.AtomicLongArray(1);
   nativeI.set(0,79);nativeL.set(0,83L);Holder<AtomicIntegerArray> holder=new Holder<>(local);
   System.out.println(local.get(0)+":"+foreign.get(0)+":"+nativeI.get(0)+":"+nativeL.get(0)+":"+(holder.get()==local));
  }
 }`,
	}, "probe.Main", "71:73:79:83:true\n")
}
