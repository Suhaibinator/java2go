package transpiler

import "testing"

func TestCampaignFamilyBoxedConsumption(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>boxed-family</artifactId><version>1</version></project>`,
		"src/main/java/review/boxed/Main.java": `package review.boxed;
public class Main {
 static class Holder<U>{U value;void set(U value){this.value=value;}<T>T pair(U owner,T result){value=owner;return result;}<U>U shadow(U value){return value;}}
 static class IntegerHolder extends Holder<Integer>{}
 static <T>Holder<T> same(Holder<T> holder){return holder;}
 static int effects;static int tick(){effects++;return 1;}
 static int take(Integer value,int ignored){return value;}
 @SuppressWarnings({"rawtypes","unchecked"})public static void main(String[] args){
  Holder<Integer> holder=same(new Holder<Integer>());holder.set(7);
  System.out.println("pair="+holder.pair(9,"paired"));
  Integer shadow=holder.shadow(11);System.out.println("shadow="+shadow);
  IntegerHolder child=new IntegerHolder();child.set(4);
  System.out.println("sum="+(holder.value+child.value));
  int taken=take(holder.value,0);System.out.println("argument="+taken);
  long wide=holder.value;System.out.println("wide="+wide);
  Holder raw=holder;raw.value="polluted";Object broad=holder.value;
  System.out.println("object="+broad);
  try{take(holder.value,tick());System.out.println("wrong-argument");}catch(ClassCastException expected){System.out.println("argument-check="+effects);}
  try{int ignored=holder.value+tick();System.out.println("wrong-arithmetic="+ignored);}catch(ClassCastException expected){System.out.println("arithmetic-check="+effects);}
  raw.value=null;System.out.println("null-object="+((Object)holder.value==null));
  try{long ignored=holder.value;System.out.println("wrong-null="+ignored);}catch(NullPointerException expected){System.out.println("null-unbox");}
 }
}
`}, "review.boxed.Main", "pair=paired\nshadow=11\nsum=13\nargument=9\nwide=9\nobject=polluted\nargument-check=0\narithmetic-check=0\nnull-object=true\nnull-unbox\n")
}
