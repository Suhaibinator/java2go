package transpiler

import "testing"

func TestHoistedLocalStaticAbruptRecursiveV2JVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>control</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static RuntimeException marker=new RuntimeException("marker");static int failing(){System.out.println("failing");throw marker;}
 static void abrupt(){class Broken{static int value=failing();}try{System.out.println(Broken.value);}catch(ExceptionInInitializerError expected){System.out.println(expected.getCause()==marker);}catch(NoClassDefFoundError expected){System.out.println("repeat-first-erroneous");}finally{System.out.println("first-finally");}try{System.out.println(Broken.value);}catch(NoClassDefFoundError expected){System.out.println("erroneous");}finally{System.out.println("second-finally");}}
 static void recursive(){class Recursive{static int before=Recursive.after;static Recursive instance=new Recursive();static int after=13;int observed=Recursive.after;static{System.out.println(before);System.out.println(instance.observed);System.out.println(after);}}System.out.println(Recursive.after);System.out.println(Recursive.instance==Recursive.instance);}
 public static void main(String[] args){abrupt();abrupt();recursive();recursive();}
}
`,
	})
}

func TestHoistedLocalStaticActiveUseOrderJVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>control</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static int events;static int step(String text,int n){events++;System.out.println(text);return n;}static int failArgument(){events++;System.out.println("argument-failed");throw new IllegalArgumentException("arg");}
 static void direct(){class Cold{static int value=step("new-init",7);int own=step("instance",8);Cold(int n){System.out.println("body:"+n);}}
 try{new Cold(failArgument());}catch(IllegalArgumentException expected){System.out.println("caught-new");}System.out.println(Cold.value);new Cold(step("new-argument",9));}
 static void method(){class Cold{static int value=step("method-init",11);static int call(int n){System.out.println("method-body");return n+value;}}
 try{Cold.call(failArgument());}catch(IllegalArgumentException expected){System.out.println("caught-method");}System.out.println(events);System.out.println(Cold.call(step("method-argument",13)));}
 public static void main(String[] args){direct();method();}
}
`,
	})
}

func TestHoistedLocalStaticNamespacesCaptureJVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>control</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static int local(){class Long{static int BYTES=67;static int SIZE=71;}int BYTES=29;System.out.println(BYTES);System.out.println(Long.SIZE);Long.BYTES+=2;return Long.BYTES;}
 static <T> void captured(T value){class Box<U>{static int count=3;T saved=value;U own;Box(U x){own=x;}Object read(){return saved;}static int bump(){return ++count;}}
 Box<String> a=new Box<String>("own");System.out.println(a.read()==value);System.out.println(a.own);System.out.println(Box.count);System.out.println(Box.bump());}
 static int one(){class Alpha{static int value=11;}return ++Alpha.value;}static int two(){class Alpha{static int value=21;}return ++Alpha.value;}
 public static void main(String[] args){System.out.println(local());System.out.println(local());captured("captured");captured("next");System.out.println(one());System.out.println(two());System.out.println(one());System.out.println(two());}
}
`,
	})
}

func TestHoistedLocalStaticNestedStaticFlowV2JVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>control</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static int calls;static int init(){System.out.println("init");return 7;}
 static int run(){outer:do{try{class Local{static int value=init();static int read(){try{return value;}finally{System.out.println("method-finally");}}}System.out.println(Local.read());if(calls++==0)return 17;}finally{System.out.println("caller-finally");}}while(false);return 19;}
 public static void main(String[] args){System.out.println(run());System.out.println(run());}
}
`,
	})
}

func TestHoistedLocalStaticQualifierMutationJVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>control</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static int effects;static IllegalStateException marker=new IllegalStateException("qualifier");static int init(){System.out.println("init");return 7;}static int dormant(){System.out.println("dormant-init");return 31;}
 static void exercise(){class Cell{static int value=init();static char glyph='Q';static int read(){return value;}}class Dormant{static int value=dormant();}class Qualifier{static Cell receiver(){effects++;return null;}static Dormant throwing(){effects++;throw marker;}}
 System.out.println(Qualifier.receiver().value);System.out.println(effects);try{System.out.println(Qualifier.throwing().value);}catch(IllegalStateException expected){System.out.println(expected==marker);}System.out.println(effects);System.out.println("dormant-unused");Cell receiver=null;System.out.println(Cell.value++);System.out.println(++Cell.value);Cell.value+=3;System.out.println(Cell.value);receiver.value=17;System.out.println(Cell.read());System.out.println(receiver.glyph);System.out.println(Dormant.value);}
 public static void main(String[] args){exercise();exercise();}
}
`,
	})
}

func TestHoistedLocalStaticStorageConstantsJVM(t *testing.T) {
	withCleanDiagnostics(t)
	campaignWrapperConstantsStrictProject(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>localstatic</groupId><artifactId>control</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static int effects;static int step(String label,int n){effects++;System.out.println(label);return n;}
 static void exercise(int captured){class Ledger{static int value=step("field",5);static final int WIDTH=java.lang.Long.BYTES;static final String LABEL="fixed";static String unset;static{System.out.println(unset==null);value+=2;}int own=captured;int read(){return own+value;}static void bump(){value++;}}
 System.out.println("before");System.out.println(Ledger.WIDTH);System.out.println(Ledger.LABEL);System.out.println(effects);System.out.println(Ledger.value);Ledger a=new Ledger();Ledger b=new Ledger();System.out.println(a.read());System.out.println(b.read());Ledger.bump();System.out.println(a.read());System.out.println(effects);}
 static void other(){class Ledger{static int value=step("other",21);}System.out.println(Ledger.value);Ledger.value++;System.out.println(Ledger.value);}
 public static void main(String[] args){exercise(10);exercise(20);other();other();}
}
`,
	})
}
