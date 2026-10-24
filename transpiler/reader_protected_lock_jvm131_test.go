package transpiler

import "testing"

// All Java and exact stdout observations were frozen from actual JDK21 before implementation.
func TestReaderProtectedLockMatrix131(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   string
	}{
		{"bridgeCollisionHeld", `package locks;import java.io.Reader;public class Main{
static class R extends Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} Object JavaReaderState(){return this;}boolean same(){return lock==this;}}
public static void main(String[]args){R r=new R();System.out.println((r.JavaReaderState()==r)+":"+r.same());}}`, "true:true\n"},
		{"construction", `package locks;import java.io.Reader;public class Main{
static class E extends Reader{boolean first=lock==this;E(){super();}public int read(char[] b,int o,int n){return -1;}public void close(){} boolean same(){return lock==this;}}
static class I extends Reader{boolean first=lock==this;I(){}public int read(char[] b,int o,int n){return -1;}public void close(){} boolean same(){return lock==this;}}
static class D extends Reader{boolean first=lock==this;public int read(char[] b,int o,int n){return -1;}public void close(){} boolean same(){return lock==this;}}
static class C extends Reader{boolean first=lock==this;C(){this(1);}C(int n){super();}public int read(char[] b,int o,int n){return -1;}public void close(){} boolean same(){return lock==this;}}
public static void main(String[]args){E e=new E();I i=new I();D d=new D();C c=new C();System.out.println(e.first+":"+e.same()+":"+i.first+":"+i.same()+":"+d.first+":"+d.same()+":"+c.first+":"+c.same());}}`, "true:true:true:true:true:true:true:true\n"},
		{"executionAndUTF16", `package locks;import java.io.Reader;public class Main{
static class R extends Reader{int calls;R(){super();}R(Object v){super(v);}public int read(char[] b,int o,int n){synchronized(lock){if(!Thread.holdsLock(lock))throw new AssertionError();calls++;if(n==0)return 0;b[o]='\uD800';return 1;}}public void close(){}Object get(){return lock;}}
public static void main(String[]args)throws Exception{R r=new R();int unit; synchronized(r){unit=r.read();}Object s=new Object();R explicit=new R(s);synchronized(s){if(explicit.get()!=s)throw new AssertionError();explicit.read();}System.out.println(unit+":"+r.calls+":"+explicit.calls+":"+(r.read((char[])null,0,0)==0));}}`, "55296:1:1:true\n"},
		{"explicitAndNull", `package locks;import java.io.Reader;public class Main{
static int calls;static int fields;static Object sentinel=new Object();static RuntimeException boom=new RuntimeException();
static Object argument(boolean fail){calls++;if(fail)throw boom;return sentinel;}
static class R extends Reader{int field=++fields;R(Object v){super(v);}R(boolean fail){super(argument(fail));}public int read(char[] b,int o,int n){return -1;}public void close(){} Object get(){return lock;}}
public static void main(String[]args){R r=new R(false);int nulls=0;try{new R((Object)null);}catch(NullPointerException e){nulls++;}Object none=null;try{new R(none);}catch(NullPointerException e){nulls++;}boolean exact=false;try{new R(true);}catch(RuntimeException e){exact=e==boom;}System.out.println((r.get()==sentinel)+":"+calls+":"+fields+":"+nulls+":"+exact);}}`, "true:2:1:2:true\n"},
		{"genericEnclosing", `package locks;import java.io.Reader;public class Main{
static class Parent<T> extends Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} boolean same(){return lock==this;}}
static class Child<T> extends Parent<T>{boolean field=lock==this;}
public static void main(String[]args){Child<String> r=new Child<String>();System.out.println(r.field+":"+r.same());}}`, "true:true\n"},
		{"hierarchy", `package locks;import java.io.Reader;public class Main{
static class Parent extends Reader{boolean base=lock==this;public int read(char[] b,int o,int n){return -1;}public void close(){} Object nativeLock(){return lock;} }
static class Child extends Parent{boolean child=lock==this;}
static class Grand extends Child{boolean grand=lock==this;}
public static void main(String[]args){Grand g=new Grand();Parent p=g;System.out.println(g.base+":"+g.child+":"+g.grand+":"+(p.nativeLock()==g));}}`, "true:true:true:true\n"},
		{"mutable", `package locks;import java.io.Reader;public class Main{
static class R extends Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} Object get(){return lock;}Object put(Object v){return lock=v;}void clear(){this.lock=null;}boolean held(){synchronized(lock){return Thread.holdsLock(lock);}}}
public static void main(String[]args){R r=new R();Object s=new Object();boolean one=r.put(s)==s&&r.get()==s;r.clear();boolean nil=r.get()==null;int npe=0;try{r.held();}catch(NullPointerException e){npe++;}boolean stayed=r.get()==null;boolean restored=r.put(r)==r&&r.get()==r;System.out.println(one+":"+nil+":"+npe+":"+stayed+":"+restored+":"+r.held());}}`, "true:true:1:true:true:true\n"},
		{"nativeCloneHeld", `package locks;import java.io.Reader;public class Main{
static class R extends Reader implements Cloneable{public int read(char[] b,int o,int n){return -1;}public void close(){} Object get(){return lock;}void put(Object v){lock=v;}R copy()throws CloneNotSupportedException{return(R)super.clone();}}
public static void main(String[]args)throws Exception{R first=new R();R clone=first.copy();boolean a=clone.get()==first;Object s=new Object();first.put(s);R other=first.copy();boolean b=other.get()==s;other.put(null);System.out.println(a+":"+b+":"+(first.get()==s)+":"+(other.get()==null));}}`, "true:true:true:true\n"},
		{"outerLambdaMonitor", `package locks;import java.io.Reader;public class Main{
Object lock=new Object();class R extends Reader{public int read(char[] b,int o,int n){return -1;}public void close(){} boolean inherited(){return lock==this;}boolean outer(){return Main.this.lock!=this;}boolean captured(){Runnable a=()->{synchronized(lock){if(!Thread.holdsLock(this))throw new AssertionError();}};a.run();class L{boolean check(){return lock==R.this;}}return new L().check();}}
public static void main(String[]args){Main o=new Main();R r=o.new R();synchronized(r){System.out.println(r.inherited()+":"+r.outer()+":"+r.captured());}}}`, "true:true:true\n"},
		{"qualifiedAssignmentOrder", `package locks;import java.io.Reader;public class Main{
static class R extends Reader{static int trace;static R target;static Object sentinel=new Object();static RuntimeException boom=new RuntimeException();public int read(char[] b,int o,int n){return -1;}public void close(){}
static R peer(boolean missing){trace=trace*10+1;return missing?null:target;}
static Object rhs(boolean fail){trace=trace*10+2;if(fail)throw boom;return sentinel;}
static void store(boolean missing,boolean fail){peer(missing).lock=rhs(fail);}
static Object value(){return(peer(false).lock=rhs(false));}
static Object nested(){return((peer(false).lock)=(peer(false).lock=rhs(false)));}
Object get(){return this.lock;}}
public static void main(String[]args){R.target=new R();R.trace=0;R.store(false,false);int a=R.trace;R.trace=0;boolean missing=false;try{R.store(true,false);}catch(NullPointerException e){missing=true;}int b=R.trace;R.trace=0;boolean throwing=false;try{R.store(true,true);}catch(RuntimeException e){throwing=e==R.boom;}int c=R.trace;R.trace=0;boolean val=R.value()==R.sentinel;int d=R.trace;R.trace=0;boolean nested=R.nested()==R.sentinel;System.out.println(a+":"+b+":"+c+":"+d+":"+R.trace+":"+missing+":"+throwing+":"+val+":"+nested+":"+(R.target.get()==R.sentinel));}}`, "12:12:12:12:112:true:true:true:true:true\n"},
		{"sourceFieldAndLocalShadow", `package locks;import java.io.Reader;public class Main{
static class R extends Reader{Object lock=new Object();public int read(char[] b,int o,int n){return -1;}public void close(){} Object nativeLock(){return super.lock;}Object own(){return this.lock;}Object parameter(Object lock){return lock;}Object local(){Object lock=new Object();return lock;}}
static class Child extends R{Object inherited(){return super.lock;}Object nativeFromParent(){return nativeLock();}}
public static void main(String[]args){Child r=new Child();Object s=new Object();System.out.println((r.own()!=r)+":"+(r.inherited()==r.own())+":"+(r.nativeFromParent()==r)+":"+(r.parameter(s)==s)+":"+(r.local()!=r.own()));}}`, "true:true:true:true:true\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string]string{"src/main/java/locks/Main.java": test.source, "pom.xml": `<project><groupId>reader</groupId><artifactId>locks</artifactId><version>1</version></project>`}
			runCampaignCompilerStrictProjectObservations(t, files, "locks.Main", []campaignCompilerProjectObservation{{name: "first", stdout: test.want}, {name: "second", stdout: test.want}, {name: "third", stdout: test.want}})
		})
	}
}
