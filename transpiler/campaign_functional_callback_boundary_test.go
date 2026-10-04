package transpiler

import "testing"

func TestCampaignFunctionalCallbackBoundaryJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-callback-boundary</artifactId><version>1</version></project>`,
		"src/main/java/review/functionboundary/Main.java": `package review.functionboundary;
import java.util.*;
import java.util.function.Function;
import java.util.concurrent.*;
import java.util.stream.Collectors;
public class Main {
 static int effects;
 static Thread expected;
 static int measure(String text){
  effects++;
  if(Thread.currentThread()!=expected)throw new IllegalStateException("execution");
  if(text.equals("boom"))throw new IllegalArgumentException("callback");
  return text.length();
 }
 static class Length implements Function<String,Integer>{public Integer apply(String text){return measure(text);}}
 public static void main(String[] args)throws Exception{
  expected=Thread.currentThread();
  int optional=Optional.of("abcd").map(Main::measure).get();
  List<String> words=new ArrayList<>();words.add("a");words.add("bbb");
  long total=words.stream().map(Main::measure).mapToLong(n->n).sum();
  Map<String,Integer> collected=words.stream().collect(Collectors.toMap(s->s,Main::measure));
  Function<String,Integer> source=new Length();
  HashMap<String,Integer> values=new HashMap<>();
  int computed=values.computeIfAbsent("cc",source);
  HashMap<String,Function<String,Integer>> stored=new HashMap<>();stored.put("mapper",source);
  boolean same=stored.get("mapper")==source;
  int invoked=stored.get("mapper").apply("x");
  String failure="missing";
  try{Optional.of("boom").map(Main::measure);}catch(IllegalArgumentException ex){failure="caught";}
  System.out.println(optional+":"+total+":"+collected.size()+":"+computed+":"+same+":"+invoked+":"+effects+":"+failure);
  Function<Thread,Boolean> identity=thread->Thread.currentThread()==thread;
  ExecutorService pool=Executors.newSingleThreadExecutor();
  try{System.out.println(pool.submit(()->{
   HashMap<Thread,Boolean> observed=new HashMap<>();
   return observed.computeIfAbsent(Thread.currentThread(),identity);
  }).get(10,TimeUnit.SECONDS));}finally{pool.shutdownNow();}
 }
}
`}, "review.functionboundary.Main", "4:4:2:2:true:1:8:caught\ntrue\n")
}
