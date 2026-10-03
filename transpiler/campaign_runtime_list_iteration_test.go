package transpiler

import "testing"

// This exercises the runtime iterator directly. Source Iterator method lowering
// remains a separate compiler prerequisite; the original join Java probes are
// still the compiler acceptance tests and are not replaced by this driver.
func TestCampaignListIteratorStructuralRevisionJVM(t *testing.T) {
	campaignStringCoreOperationOracle(t, "ListRevisions", `import java.util.*;
public class ListRevisions {
 interface Action {void run();}
 static String outcome(Action a){try{a.run();return "none";}catch(RuntimeException e){return e.getClass().getSimpleName();}}
 public static void main(String[] args){
  ArrayList<String> a=new ArrayList<>();a.add("x");Iterator<String> i=a.iterator();
  System.out.println("add-empty="+a.addAll(Collections.emptyList())+":"+i.hasNext()+":"+outcome(()->i.next()));
  ArrayList<String> b=new ArrayList<>();Iterator<String> j=b.iterator();b.clear();
  System.out.println("clear-empty="+j.hasNext()+":"+outcome(()->j.next()));
  ArrayList<String> c=new ArrayList<>();c.add("x");Iterator<String> k=c.iterator();c.clear();
  System.out.println("clear="+k.hasNext()+":"+outcome(()->k.next()));
  String[] array={"a","old"};List<String> fixed=Arrays.asList(array);Iterator<String> f=fixed.iterator();
  System.out.println("fixed-empty="+fixed.addAll(Collections.emptyList())+":"+f.next());array[1]="changed";
  System.out.println("fixed-live="+f.next()+":"+f.hasNext()+":"+outcome(()->f.next()));
  List<String> empty=Arrays.asList(new String[0]);Iterator<String> e=empty.iterator();empty.clear();
  System.out.println("fixed-clear-empty="+e.hasNext()+":"+outcome(()->e.next()));
  Iterator<String> unchanged=fixed.iterator();System.out.println("fixed-failure="+outcome(()->fixed.add("bad"))+":"+unchanged.next());
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func outcome(action func())(result string){result="none";defer func(){if failure:=recover();failure!=nil {if throwable,ok:=failure.(j.Throwable);ok {result=throwable.ThrowableTypeName()}else{panic(failure)}}}();action();return}
func main(){
 e:=j.NewExecution();a:=j.NewListFrom("x");i:=j.IterableIteratorExecution(e,a);added:=a.AddAll(j.NewList[string]())
 fmt.Printf("add-empty=%t:%t:%s\n",added,j.IteratorHasNextExecution(e,i),outcome(func(){j.IteratorNextExecution(e,i)}))
 b:=j.NewList[string]();bi:=j.IterableIteratorExecution(e,b);b.Clear();fmt.Printf("clear-empty=%t:%s\n",j.IteratorHasNextExecution(e,bi),outcome(func(){j.IteratorNextExecution(e,bi)}))
 c:=j.NewListFrom("x");ci:=j.IterableIteratorExecution(e,c);c.Clear();fmt.Printf("clear=%t:%s\n",j.IteratorHasNextExecution(e,ci),outcome(func(){j.IteratorNextExecution(e,ci)}))
 array:=j.ReferenceArrayLiteral(j.StringTypeID,"a","old");fixed:=j.AsListArray[string](array,j.StringTypeID);f:=j.IterableIteratorExecution(e,fixed)
 fmt.Printf("fixed-empty=%t:%s\n",fixed.AddAll(j.NewList[string]()),j.IteratorNextExecution(e,f));j.ReferenceArraySet(array,1,"changed")
 fmt.Printf("fixed-live=%s:%t:%s\n",j.IteratorNextExecution(e,f),j.IteratorHasNextExecution(e,f),outcome(func(){j.IteratorNextExecution(e,f)}))
 empty:=j.AsListArray[string](j.NewReferenceArray(0,j.StringTypeID),j.StringTypeID);ei:=j.IterableIteratorExecution(e,empty);empty.Clear();fmt.Printf("fixed-clear-empty=%t:%s\n",j.IteratorHasNextExecution(e,ei),outcome(func(){j.IteratorNextExecution(e,ei)}))
 unchanged:=j.IterableIteratorExecution(e,fixed);fmt.Printf("fixed-failure=%s:%s\n",outcome(func(){fixed.Add("bad")}),j.IteratorNextExecution(e,unchanged))
}
`)
}
