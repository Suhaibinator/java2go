package transpiler

import "testing"

// This is the frozen business-family failure's cast shape: a named universal
// factory casts a concrete specialized subclass directly to its generic base.
func TestCampaignFamilyReview08SuperclassCast(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>superclass-cast</artifactId><version>1</version></project>`,
		"src/main/java/review/cast/Main.java": `package review.cast;
abstract class Adapter<T>{public abstract T read();public abstract void write(T value);}
class NumberAdapter extends Adapter<Number>{private Number value=7;int entered;public Integer read(){return Integer.valueOf(value.intValue());}public void write(Number value){entered++;this.value=value;}}
interface Factory{<T> Adapter<T> create();}
class NamedFactory implements Factory{private final NumberAdapter bridge;NamedFactory(NumberAdapter bridge){this.bridge=bridge;}@SuppressWarnings("unchecked") public <T> Adapter<T> create(){return (Adapter<T>)bridge;}}
class Other{}
public class Main{@SuppressWarnings({"unchecked","rawtypes"}) public static void main(String[] args){
 NumberAdapter concrete=new NumberAdapter();Factory factory=new NamedFactory(concrete);Adapter<Number> view=factory.create();
 System.out.println("identity="+((Object)view==(Object)concrete)+":"+(view.getClass()==concrete.getClass()));
 System.out.println("value="+view.read().intValue());view.write(23);System.out.println("updated="+concrete.read().intValue());
 Adapter raw=view;try{raw.write("wrong");System.out.println("wrong-bridge");}catch(ClassCastException expected){System.out.println("bridge="+concrete.entered);}
 Factory absent=new NamedFactory(null);System.out.println("null="+(absent.<String>create()==null));
 Object unrelated=new Other();try{Adapter<?> wrong=(Adapter<?>)unrelated;System.out.println("wrong-cast="+wrong);}catch(ClassCastException expected){System.out.println("wrong=checked");}
 }}
`}, "review.cast.Main", "identity=true:true\nvalue=7\nupdated=23\nbridge=1\nnull=true\nwrong=checked\n")
}
