package transpiler

import "testing"

// A local implementation joins both source interface families even though it
// is absent from the global named-class graph. Both erased descriptors must be
// usable on the same allocation, and both narrow bridges check before entry.
func TestCampaignFamilyLocalMultipleInterfaces(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>local-interfaces</artifactId><version>1</version></project>`,
		"src/main/java/review/multiple/Main.java": `package review.multiple;
interface Left<T>{T read();void write(T value);}
interface Right<T>{T get();void put(T value);}
interface LeftFactory{<T>Left<T> create();}
interface RightFactory{<T>Right<T> create();}
public class Main {
 @SuppressWarnings({"unchecked","rawtypes"})public static void main(String[] args){
  class Dual implements Left<String>,Right<String>{
   String value="seed";int calls;
   public String read(){return value;}public void write(String next){calls++;value=next;}
   public String get(){return value;}public void put(String next){calls++;value=next;}
  }
  Dual concrete=new Dual();
  LeftFactory lf=new LeftFactory(){public <T>Left<T> create(){return (Left<T>)concrete;}};
  RightFactory rf=new RightFactory(){public <T>Right<T> create(){return (Right<T>)concrete;}};
  Left<String> left=lf.create();Right<String> right=rf.create();
  System.out.println("initial="+left.read()+":"+right.get()+":"+((Object)left==(Object)right));
  left.write("alpha");System.out.println("left="+right.get()+":"+concrete.calls);
  right.put("beta");System.out.println("right="+left.read()+":"+concrete.calls);
  Left rawLeft=left;Right rawRight=right;
  try{rawLeft.write(17);System.out.println("wrong-left");}catch(ClassCastException expected){System.out.println("left-check="+concrete.calls);}
  try{rawRight.put(23);System.out.println("wrong-right");}catch(ClassCastException expected){System.out.println("right-check="+concrete.calls);}
  System.out.println("retained="+left.read()+":"+right.get());
 }
}
`}, "review.multiple.Main", "initial=seed:seed:true\nleft=alpha:1\nright=beta:2\nleft-check=2\nright-check=2\nretained=beta:beta\n")
}
