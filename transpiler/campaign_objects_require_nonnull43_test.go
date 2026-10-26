package transpiler

import "testing"

func TestCampaignObjectsRequireNonNull43IdentityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import static java.util.Objects.requireNonNull;public class Main {static class Item{}public static void main(String[] args){Item item=new Item();Object erased=item;boolean one=requireNonNull(item)==item,two=java.util.Objects.requireNonNull(erased)==item,npe=false;try{Item unused=Main.<Item>check(null);}catch(NullPointerException expected){npe=expected.getMessage()==null;}Integer boxed=requireNonNull(7);System.out.print(one+":"+two+":"+npe+":"+boxed.intValue());}static <T> T check(T value){return requireNonNull(value);}}`,
	}, "probe.Main", "true:true:true:7")
}
func TestCampaignObjectsRequireNonNull43MessageJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import java.util.Objects;public class Main {static int calls;static String message(){calls++;return "detail";}public static void main(String[] args){Object value=new Object();boolean same=Objects.requireNonNull(value,message())==value,detail=false,empty=false,absent=false;try{Objects.requireNonNull((Object)null,message());}catch(NullPointerException e){detail=e.getMessage().equals("detail");}try{Objects.requireNonNull((Object)null,"");}catch(NullPointerException e){empty=e.getMessage().equals("");}try{Objects.requireNonNull((Object)null,(String)null);}catch(NullPointerException e){absent=e.getMessage()==null;}System.out.print(same+":"+detail+":"+empty+":"+absent+":"+calls);}}`,
	}, "probe.Main", "true:true:true:true:2")
}
func TestCampaignObjectsRequireNonNull43SupplierJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import java.util.Objects;import java.util.function.Supplier;public class Main {static class Message implements Supplier<String>{int calls;boolean same;Thread expected;boolean fail;public String get(){calls++;same=Thread.currentThread()==expected;if(fail)throw new IllegalArgumentException("supplier");return null;}}public static void main(String[] args)throws Exception{Message supplier=new Message();StringBuilder out=new StringBuilder();Thread creator=Thread.currentThread();Thread worker=new Thread(()->{supplier.expected=Thread.currentThread();Object value=new Object();boolean same=Objects.requireNonNull(value,supplier)==value;boolean lazy=supplier.calls==0;boolean noSupplier=Objects.requireNonNull(value,(Supplier<String>)null)==value;boolean nullMessage=false,nullSupplier=false,thrown=false;try{Objects.requireNonNull((Object)null,supplier);}catch(NullPointerException e){nullMessage=e.getMessage()==null;}try{Objects.requireNonNull((Object)null,(Supplier<String>)null);}catch(NullPointerException e){nullSupplier=e.getMessage()==null;}supplier.fail=true;try{Objects.requireNonNull((Object)null,supplier);}catch(IllegalArgumentException e){thrown=e.getMessage().equals("supplier");}out.append(same).append(':').append(lazy).append(':').append(noSupplier).append(':').append(nullMessage).append(':').append(nullSupplier).append(':').append(thrown).append(':').append(supplier.calls).append(':').append(supplier.same&&Thread.currentThread()!=creator);});worker.start();worker.join();System.out.print(out.toString());}}`,
	}, "probe.Main", "true:true:true:true:true:true:2:true")
}
func TestCampaignObjectsRequireNonNull43SourceShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/probe/Objects.java": `package probe;public class Objects {public static int requireNonNull(int value){return value+10;}}`,
		"src/main/java/probe/Main.java":    `package probe;public class Main {public static void main(String[] args){String value="ok";System.out.print(Objects.requireNonNull(7)+":"+(java.util.Objects.requireNonNull(value)==value));}}`,
	}, "probe.Main", "17:true")
}
