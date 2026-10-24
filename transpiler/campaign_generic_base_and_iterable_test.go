package transpiler

import "testing"

func TestCampaignCollectionArgumentToIterable(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>collection-argument</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example; import java.util.*;
 public class Main {
 public boolean balanced(Iterable<String> values, Map<String,Integer> counts) {int total=0;for(String value:values)total+=counts.get(value);return total==3;}
 public static void main(String[] args){Map<String,String> values=new LinkedHashMap<>();values.put("a","x");values.put("b","y");Map<String,Integer> counts=new HashMap<>();counts.put("x",1);counts.put("y",2);System.out.println(new Main().balanced(values.values(),counts));}
 }`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "true\n")
}

func TestCampaignGenericAbstractBaseFactory(t *testing.T) {
	files := map[string]string{
		"pom.xml":                             `<project><groupId>example</groupId><artifactId>abstract-base</artifactId><version>1</version></project>`,
		"src/main/java/base/Document.java":    `package base; import derived.Shipment; public abstract class Document<K>{private final K id;protected Document(K id){this.id=id;}public final K id(){return id;}public abstract String state();public static Document<String> create(String id){return new Shipment(id);}public static Document<String> optional(Shipment value){return value;} }`,
		"src/main/java/derived/Shipment.java": `package derived;import base.Document;public final class Shipment extends Document<String>{public Shipment(String id){super(id);}public String state(){return "ready";}}`,
		"src/main/java/example/Main.java":     `package example;import base.Document;import derived.Shipment;public class Main {public static void main(String[] args){Document<String> value=Document.create("item");System.out.println(value.id()+":"+value.state());System.out.println(((Shipment)value).state());System.out.println(Document.optional(null)==null);System.out.println(value instanceof Shipment);java.util.function.Supplier<String> bound=value::state;System.out.println(bound.get());java.util.function.Function<Document<String>,String> unbound=Document<String>::state;System.out.println(unbound.apply(value));}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "item:ready\nready\ntrue\ntrue\nready\nready\n")
}
