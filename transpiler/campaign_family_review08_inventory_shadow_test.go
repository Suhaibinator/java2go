package transpiler

import "testing"

// A method-local Slot must not change the meaning of the simple Slot name in
// Factory.java or Observer.java when whole-project inventory visits them.
func TestCampaignFamilyReview08InventoryShadow(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml":                                   `<project><groupId>review</groupId><artifactId>inventory-shadow</artifactId><version>1</version></project>`,
		"src/main/java/review/shadow/Slot.java":     `package review.shadow;public interface Slot<T>{T get();void set(T value);}`,
		"src/main/java/review/shadow/Cell.java":     `package review.shadow;public class Cell<T> implements Slot<T>{private T value;public Cell(T value){this.value=value;}public T get(){return value;}public void set(T value){this.value=value;}}`,
		"src/main/java/review/shadow/Factory.java":  `package review.shadow;public interface Factory{<T> Slot<T> make();}`,
		"src/main/java/review/shadow/Observer.java": `package review.shadow;public class Observer{public static String read(Slot<String> value){return value.get().toUpperCase();}}`,
		"src/main/java/review/shadow/AHost.java": `package review.shadow;
public class AHost{
 static String localLabel(){class Slot{String label(){return "local";}}Slot temporary=new Slot();return temporary.label();}
 @SuppressWarnings("unchecked") public static void main(String[] args){
 String label=localLabel();Cell<String> shared=new Cell<String>("seed");
 Factory factory=new Factory(){public <T> review.shadow.Slot<T> make(){return (review.shadow.Slot<T>)shared;}};
 review.shadow.Slot<String> first=factory.make();review.shadow.Slot<String> second=factory.make();
 System.out.println("label="+label);System.out.println("value="+Observer.read(first));
 System.out.println("same="+((Object)first==(Object)second));second.set("next");String next=first.get();System.out.println("updated="+next);
 }}
`}, "review.shadow.AHost", "label=local\nvalue=SEED\nsame=true\nupdated=next\n")
}
