package transpiler

import "testing"

func TestCampaignFamilyReview08LocalImplements(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>local-implements</artifactId><version>1</version></project>`,
		"src/main/java/review/local/Main.java": `package review.local;
interface Slot<T>{T read();void write(T value);}
interface Factory{<T> Slot<T> create();}
public class Main{@SuppressWarnings({"unchecked","rawtypes"}) public static void main(String[] args){
 class Local implements Slot<String>{String value="seed";int entered;public String read(){return value;}public void write(String next){entered++;value=next;}}
 Local concrete=new Local();Factory factory=new Factory(){public <T> Slot<T> create(){return (Slot<T>)concrete;}};
 Slot<String> typed=factory.create();Slot raw=typed;
 System.out.println("same="+((Object)typed==(Object)concrete));
 typed.write("changed");System.out.println("value="+typed.read()+":"+concrete.entered);
 try{raw.write(17);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("bridge="+concrete.entered);}
 System.out.println("retained="+typed.read());
 }}
`}, "review.local.Main", "same=true\nvalue=changed:1\nbridge=1\nretained=changed\n")
}
