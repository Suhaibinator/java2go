package transpiler

import "testing"

func TestCampaignFunctionalNullMapArgumentJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-null</artifactId><version>1</version></project>`,
		"src/main/java/review/functionnull/Main.java": `package review.functionnull;
import java.util.HashMap;
import java.util.function.Function;
public class Main {
 static int effects;
 static String argument(){effects++;return "present";}
 public static void main(String[] args){
  Function<String,String> absent=null;
  Object alias=absent;
  HashMap<String,String> map=new HashMap<>();map.put("present","kept");
  System.out.println("null="+(absent==null)+":"+(alias==null));
  try {map.computeIfAbsent(argument(),absent);System.out.println("missing-npe");}
  catch(NullPointerException expected){System.out.println("npe="+effects+":"+map.get("present"));}
 }
}
`}, "review.functionnull.Main", "null=true:true\nnpe=1:kept\n")
}

func TestCampaignFunctionalNullApplyJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-null-apply</artifactId><version>1</version></project>`,
		"src/main/java/review/functionapply/Main.java": `package review.functionapply;
public class Main {
 static int effects;
 static String argument(){effects++;return "value";}
 public static void main(String[] args){
  java.util.function.Function<String,String> absent=null;
  try {absent.apply(argument());System.out.println("missing-npe");}
  catch(NullPointerException expected){System.out.println("npe="+effects);}
 }
}
`}, "review.functionapply.Main", "npe=1\n")
}

func TestCampaignFunctionalObjectIdentityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-identity</artifactId><version>1</version></project>`,
		"src/main/java/review/functionidentity/Main.java": `package review.functionidentity;
import java.util.function.Function;
import java.util.HashMap;
public class Main {
 public static void main(String[] args){
  Function<String,String> mapping=new Function<>(){public String apply(String text){return text+"!";}};
  Function<String,String> same=mapping;Object object=mapping;
  Function<String,String> other=new Function<>(){public String apply(String text){return text+"!";}};
  HashMap<String,String> map=new HashMap<>();
  System.out.println("identity="+(mapping==same)+":"+(object==mapping)+":"+(mapping==other));
  System.out.println("calls="+mapping.apply("direct")+":"+map.computeIfAbsent("mapped",mapping));
 }
}
`}, "review.functionidentity.Main", "identity=true:true:false\ncalls=direct!:mapped!\n")
}

func TestCampaignFunctionalCallerExecutionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-execution</artifactId><version>1</version></project>`,
		"src/main/java/review/functionexecution/Main.java": `package review.functionexecution;
import java.util.concurrent.*;
import java.util.function.Function;
public class Main {
 public static void main(String[] args)throws Exception{
  Function<Thread,Boolean> mapping=thread -> Thread.currentThread()==thread;
  Thread creator=Thread.currentThread();
  ExecutorService pool=Executors.newSingleThreadExecutor();
  try{
   System.out.println(pool.submit(()->mapping.apply(Thread.currentThread())).get(10,TimeUnit.SECONDS)+":"+pool.submit(()->mapping.apply(creator)).get(10,TimeUnit.SECONDS));
  }
  finally{pool.shutdownNow();}
 }
}
`}, "review.functionexecution.Main", "true:false\n")
}

func TestCampaignFunctionalCanonicalOwnerJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-owner</artifactId><version>1</version></project>`,
		"src/main/java/review/functionowner/Main.java": `package review.functionowner;
public class Main {
 static class Function<A,B>{String label(){return "source";}}
 public static void main(String[] args){
  Function<String,String> source=new Function<>();
  java.util.function.Function<String,String> external=null;
  System.out.println(source.label()+":"+(external==null));
 }
}
`}, "review.functionowner.Main", "source:true\n")
}

func TestCampaignFunctionalReceiverStagingJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-receiver-staging</artifactId><version>1</version></project>`,
		"src/main/java/review/functionstaging/Main.java": `package review.functionstaging;
import java.util.function.Function;
public class Main {
 static Function<String,String> current;
 static Function<String,String> replacement;
 static int effects;
 static String replace(){effects++;current=replacement;return "value";}
 static String clear(){effects++;current=null;return "value";}
 public static void main(String[] args){
  Function<String,String> first=new Function<>(){public String apply(String text){return "first:"+text;}};
  replacement=new Function<>(){public String apply(String text){return "second:"+text;}};
  current=first;
  System.out.println(current.apply(replace())+":"+(current==replacement));
  current=null;
  try{current.apply(replace());System.out.println("missing-npe");}
  catch(NullPointerException expected){System.out.println("null-receiver:"+(current==replacement)+":"+effects);}
  current=first;
  System.out.println(current.apply(clear())+":"+(current==null)+":"+effects);
 }
}
`}, "review.functionstaging.Main", "first:value:true\nnull-receiver:true:2\nfirst:value:true:3\n")
}

func TestCampaignFunctionalInheritedExecutionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-inherited-execution</artifactId><version>1</version></project>`,
		"src/main/java/review/functioninherited/Main.java": `package review.functioninherited;
import java.util.concurrent.*;
import java.util.function.Function;
public class Main {
 static class Base implements Function<Thread,String> {
  public String apply(Thread __java2goExecution){return Thread.currentThread()==__java2goExecution?"base":"wrong";}
 }
 static class Inherited extends Base {
  public String apply(Integer value){return "integer";}
 }
 static class Overriding extends Inherited {
  @Override public String apply(Thread __java2goExecution){return Thread.currentThread()==__java2goExecution?"override":"wrong";}
 }
 public static void main(String[] args)throws Exception{
  Function<Thread,String> inherited=new Inherited();
  Function<Thread,String> overridden=new Overriding();
  ExecutorService pool=Executors.newSingleThreadExecutor();
  try{
   System.out.println(pool.submit(()->inherited.apply(Thread.currentThread())+":"+overridden.apply(Thread.currentThread())).get(10,TimeUnit.SECONDS));
  }finally{pool.shutdownNow();}
 }
}
`}, "review.functioninherited.Main", "base:override\n")
}
