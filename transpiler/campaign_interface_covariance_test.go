package transpiler

import "testing"

func TestCampaignGenericInterfaceCovariantDefault(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>covariant-default</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.util.function.Supplier;
interface Value<T> extends Supplier<T>{T getValue(); void setValue(T value);default T get(){return getValue();}}
class Counter implements Value<Number>{int n=4;public Integer getValue(){return Integer.valueOf(n++);}public void setValue(Number value){n=value.intValue();}}
public class Main{public static void main(String[] args){Counter exact=new Counter();Value<Number> wide=exact;Integer narrow=exact.getValue();System.out.println(narrow);System.out.println(wide.getValue().intValue());System.out.println(wide.get().intValue());wide.setValue(9);System.out.println(exact.getValue());}}
`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "4\n5\n6\n9\n")
}
