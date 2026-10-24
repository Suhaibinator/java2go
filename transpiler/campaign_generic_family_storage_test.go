package transpiler

import "testing"

func TestCampaignGenericFamilyStorageDependencies(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>family-storage</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
class Cell<T>{T value;Link<T> next;Cell(T value){this.value=value;}T read(){return value;}void write(T value){this.value=value;}}
class Link<U>{Cell<U> previous;Link(Cell<U> previous){this.previous=previous;}}
class Adapter<V>{Cell<V> cell;Adapter(Cell<V> value){cell=value;}Cell<V> cell(){return cell;}}
interface Factory{<S> Adapter<S> create();}
public class Main{@SuppressWarnings({"unchecked","rawtypes"}) public static void main(String[] args){
 Cell<String> stored=new Cell<String>("seed");stored.next=new Link<String>(stored);Adapter<String> shared=new Adapter<String>(stored);
 Factory factory=new Factory(){public <T> Adapter<T> create(){return (Adapter<T>)shared;}};
 Adapter<String> first=factory.create();Adapter<Integer> other=factory.create();
 System.out.println("same="+((Object)first==(Object)other)+":"+(first.cell().next.previous==stored));
 Cell raw=other.cell();raw.write(17);System.out.println("broad="+raw.read());
 try{String value=first.cell().read();System.out.println("wrong="+value);}catch(ClassCastException expected){System.out.println("delayed");}
 raw.write("updated");System.out.println(first.cell().next.previous.read());
 }}
`}, "example.Main", "same=true:true\nbroad=17\ndelayed\nupdated\n")
}
