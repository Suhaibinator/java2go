package transpiler

import "testing"

func TestAbstractListDefaultArrayConversionEffectsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-array</artifactId><version>1</version></project>`,
		"src/main/java/probe/ArrayMain.java": `package probe;import java.util.*;
 public class ArrayMain{
  static String[] target;static class Values extends AbstractList<Object>{int sizes,gets;boolean sawStore;
   public int size(){sizes++;return 2;}public Object get(int i){gets++;if(i==1){sawStore="a".equals(target[0]);return Integer.valueOf(7);}return "a";}
  }
  public static void main(String[]args){Values values=new Values();target=new String[]{"x","y","z"};
   try{values.toArray(target);System.out.println("bad-store");}catch(ArrayStoreException e){System.out.println("store:"+target[0]+":"+target[1]+":"+values.gets+":"+values.sawStore);}
   values.sizes=0;try{values.toArray((Object[])null);System.out.println("bad-null");}catch(NullPointerException e){System.out.println("null-size:"+values.sizes);}
  }
 }`}, "probe.ArrayMain", "store:a:y:2:true\nnull-size:1\n")
}
