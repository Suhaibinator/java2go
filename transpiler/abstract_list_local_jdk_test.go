package transpiler

import "testing"

func TestAbstractListLateLocalProducerKeepsCallerExecutionJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-local</artifactId><version>1</version></project>`,
		"src/main/java/probe/LocalMain.java": `package probe;import java.util.*;
  public class LocalMain{
   static Object read(List<Object> value){return value.get(0);}
   public static void main(String[]args){Object monitor=new Object();
    class Local extends AbstractList<Object>{final Object lock;Local(Object lock){this.lock=lock;}public int size(){return 1;}public Object get(int index){if(!Thread.holdsLock(lock))throw new IllegalStateException("lost execution");return "local";}}
    List<Object> value=new Local(monitor);synchronized(monitor){System.out.println(read(value));}System.out.println(Thread.holdsLock(monitor));
   }
  }`}, "probe.LocalMain", "local\nfalse\n")
}
