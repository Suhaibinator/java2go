package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestThreadSourceFocusedObserverOriginDiagnostic(t *testing.T) {
	source := `package threadprobe;
public class Main {
 static final ThreadLocal<Integer> local=new ThreadLocal<>();
 static int targetCalls,sourceCalls,startCalls,initCalls;
 static Thread caller;
 static void target(){targetCalls++;if(Thread.currentThread()!=caller||local.get()!=41)throw new AssertionError("target-caller");}
 static class Base extends Thread {
  Base(){super();}
  Base(Runnable target,String name){super(target,name);}
 }
 static class Child extends Base {
  Child(){super(Main::target,"child");}
  public void run(){sourceCalls++;if(Thread.currentThread()!=caller)throw new AssertionError("direct-caller");super.run();}
  public boolean isInterrupted(){return !super.isInterrupted();}
  public void interrupt(){super.interrupt();}
 }
 static class NoRun extends Thread {NoRun(){super();}}
 static class StartOverride extends Thread {
  boolean launch;
  public void start(){startCalls++;if(Thread.currentThread()!=caller)throw new AssertionError("start-caller");if(launch)super.start();}
  public void run(){if(Thread.currentThread()!=this||local.get()!=null)throw new AssertionError("worker");sourceCalls++;}
 }
 static class Middle extends Thread {int calls;public void interrupt(){calls++;}}
 static class Grand extends Middle {public void interrupt(){super.interrupt();}}
 static class CtorBase extends Thread {int observed;CtorBase()throws Exception{start();join();}}
 static class CtorChild extends CtorBase {int value=7;CtorChild()throws Exception{super();}public void run(){observed=value;}}
 static class Delegated extends Thread {int value=++initCalls;Delegated(){this(1);}Delegated(int x){super();}}
 static class Collision extends Thread {int Thread=8;public void Run(){throw new AssertionError("capitalRun");}}
 static Thread identity(Thread value){return value;}
 static Runnable runnable(Runnable value){return value;}
 public static void main(String[]args)throws Exception{
  caller=Thread.currentThread();local.set(41);
  Child child=new Child();Thread base=child;base.run();child.run();runnable(child).run();
  if(targetCalls!=3||sourceCalls!=3||!child.isInterrupted())throw new AssertionError("run");
  base.interrupt();if(child.isInterrupted()||base.isInterrupted())throw new AssertionError("observer");
  Thread[] values={child};if(identity(child)!=child||values[0]!=child||(Object)base!=(Object)child||!base.getName().equals("child"))throw new AssertionError("views");
  NoRun empty=new NoRun();empty.run();((Runnable)empty).run();
  StartOverride worker=new StartOverride();Thread w=worker;w.start();if(worker.isAlive())throw new AssertionError("no-super-start");worker.launch=true;w.start();w.join();
  Grand grand=new Grand();Thread g=grand;g.interrupt();if(grand.calls!=1||grand.isInterrupted())throw new AssertionError("source-super");
  CtorChild early=new CtorChild();if(early.observed!=0||early.value!=7)throw new AssertionError("early-fields");
  Delegated once=new Delegated();if(once.value!=1||initCalls!=1)throw new AssertionError("this");
  Collision collision=new Collision();collision.run();if(collision.Thread!=8)throw new AssertionError("selector");
  Thread absent=identity(null);if(absent!=null)throw new AssertionError("null");
  System.out.println("thread-source:"+targetCalls+":"+sourceCalls+":"+startCalls+":"+early.observed+":"+once.value);
 }
}
`
	h := setupParseHelper(t, source)
	ctx := h.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Main")
	for _, m := range ctx.currentClass.Methods {
		if m.OriginalName == "main" {
			ctx.localScope = m
		}
	}
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" && (n.ChildByFieldName("name").Content(h.File.Source) == "isInterrupted" || n.ChildByFieldName("name").Content(h.File.Source) == "run") {
			o := n.ChildByFieldName("object")
			if o != nil && (o.Content(h.File.Source) == "base" || o.Content(h.File.Source) == "runnable(child)") {
				var origin inferredJavaTypeOrigin
				typ, ok := inferExprJavaType(o, ctx, h.File.Source, &origin)
				key, keyOK := intrinsicMethodReceiverTypeName(o, "run", ctx, h.File.Source)
				if !ok || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
					t.Fatal("canonical declared receiver origin missing", origin)
				}
				expected := "java.lang.Thread"
				if o.Content(h.File.Source) == "runnable(child)" {
					expected = "java.lang.Runnable"
				}
				if origin.externalName != expected || !keyOK {
					t.Fatalf("declared owner=%s key=%s/%v want%s", origin.externalName, key, keyOK, expected)
				}
				t.Logf("object=%s actual=%s/%v origin=%+v key=%s/%v native=%v plan=%+v", o.Content(h.File.Source), typ, ok, origin, key, keyOK, threadNativeStateReceiverApplicable(n, ctx, h.File.Source), threadSourceSelectInvocation(n, h.File.Source, ctx))
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
}
