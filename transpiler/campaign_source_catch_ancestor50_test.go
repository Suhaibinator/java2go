package transpiler

import "testing"

func TestCampaignSourceCatchAncestorViewJVM(t *testing.T) {
 runCampaignCompilerStrictProjectOracle(t, map[string]string{
 "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>catchviewancestor</artifactId><version>1</version></project>",
 "src/main/java/probe/catchviewancestor/Main.java": `package probe.catchviewancestor;
public final class Main {
 static String trace="";
 static class LocalFailure extends RuntimeException {
  final String stage;int calls;boolean held;
  LocalFailure(String message,String stage,Throwable cause){super(message,cause);this.stage=stage;}
  String stage(){calls++;held=Thread.holdsLock(this);trace+="m";return stage;}
 }
 static final class ChildFailure extends LocalFailure {
  ChildFailure(Throwable cause){super("child-message","base-stage",cause);}
  @Override String stage(){calls++;held=Thread.holdsLock(this);trace+="v";return "child-stage";}
 }
 public static void main(String[] args){
  Throwable cause=new IllegalArgumentException("cause");
  LocalFailure original=new LocalFailure("root","stage",cause);RuntimeException erased=original;Object alias=original;
  synchronized(original){
   try{throw erased;}catch(LocalFailure caught){trace+="c";String stage=caught.stage();System.out.println((caught==original)+":"+(caught==alias)+":"+(caught.getCause()==cause)+":"+caught.getMessage()+":"+stage+":"+caught.calls+":"+caught.held);}finally{trace+="f";}
  }
  LocalFailure empty=new LocalFailure(null,null,null);
  try{throw empty;}catch(LocalFailure caught){trace+="z";System.out.println((caught==empty)+":"+(caught.getMessage()==null)+":"+(caught.getCause()==null)+":"+(caught.stage()==null)+":"+caught.calls);}finally{trace+="g";}
  ChildFailure child=new ChildFailure(cause);LocalFailure base=child;
  synchronized(child){try{throw child;}catch(LocalFailure caught){trace+="a";System.out.println((caught==child)+":"+(caught==base)+":"+(caught.getCause()==cause)+":"+caught.stage()+":"+caught.calls+":"+caught.held);}finally{trace+="b";}}
  boolean rethrown=false;
  try{try{throw original;}catch(LocalFailure caught){trace+="i";throw caught;}finally{trace+="n";}}catch(RuntimeException caught){trace+="o";rethrown=caught==original;}finally{trace+="h";}
  System.out.println(rethrown+":"+original.calls+":"+empty.calls+":"+trace);
 }
}
`,
 }, "probe.catchviewancestor.Main", "true:true:true:root:stage:1:true\ntrue:true:true:true:1\ntrue:true:true:child-stage:1:true\ntrue:1:1:cmfzmgavbinoh\n")
}
