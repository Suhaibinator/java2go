package transpiler

import "testing"

func TestCampaignPatternExpressionFlowJVM(t *testing.T) {
 runCampaignCompilerStrictProjectOracle(t, map[string]string{
 "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>pattern</artifactId><version>1</version></project>",
 "src/main/java/probe/pattern/Main.java": `package probe.pattern;
public final class Main {
 static final class Item { final String text; final int count; Item(String t,int c){text=t;count=c;} }
 static int subjects;static int checks;static String trace="";
 static Object subject(Object value){subjects++;trace+="s";return value;}
 static String selected(String text){trace+="["+text+"]";return text;}
 static boolean check(Item item){checks++;trace+="r";return item.count>0;}
 static String describe(Object value){return subject(value) instanceof Item bound ? selected(bound.text) : selected("miss");}
 static String reversed(Object value){return !(subject(value) instanceof Item bound) ? selected("miss") : selected(bound.text);}
 static boolean matches(Object value){return subject(value) instanceof Item bound && check(bound) && check(bound);}
 static boolean accepts(Object value){return !(subject(value) instanceof Item bound) || check(bound);}
 static String guarded(Object value){if(subject(value) instanceof Item bound && check(bound))return bound.text;else return "miss";}
 static String unused(Object value){return subject(value) instanceof Item ok ? "yes" : "no";}
 public static void main(String[] args){
  Item positive=new Item("hit",1);Item negative=new Item("zero",0);
  System.out.println(describe(positive)+":"+describe(null)+":"+describe("other")+":"+reversed(positive)+":"+reversed(null));
  System.out.println(matches(positive)+":"+matches(negative)+":"+matches(null)+":"+accepts(positive)+":"+accepts(null));
  System.out.println(guarded(positive)+":"+guarded(negative)+":"+guarded(null)+":"+unused(positive)+":"+unused(null));
  System.out.println(subjects+":"+checks+":"+trace);
 }
}
`,
 }, "probe.pattern.Main", "hit:miss:miss:hit:miss\ntrue:false:false:true:true\nhit:miss:miss:yes:no\n15:6:s[hit]s[miss]s[miss]s[hit]s[miss]srrsrssrssrsrsss\n")
}
