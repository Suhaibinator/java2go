package transpiler

import "testing"

// Additive controls only. The frozen full original default/named22:33 driver
// and TestGeneratedInstanceIdentifierStrictJVM/privateNominalClone stay exact.
func TestSourceInstanceReceiverAdditionalJDK21(t *testing.T) {
	for _, test := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"genericHelper", map[string]string{"src/main/java/receivers/Main.java": `package receivers;class G1<T>{int code=9;<U>G1<T> identity(U g1_0,int g1_1){return (G1<T>)this;}}class g1_2{}public class Main{public static void main(String[]args){G1<String> first=new G1<String>();System.out.println((first==first.identity("x",1))+":"+first.code);}}`}, "true:9\n"},
		{"lambdaOuterAndMonitor", map[string]string{"src/main/java/receivers/Main.java": `package receivers;class Root implements Cloneable{int base=4;}class A1 extends Root{int value=5;boolean locked(){return Thread.holdsLock(this);}synchronized A1 copy()throws CloneNotSupportedException{if(!locked())throw new AssertionError();Runnable action=()->{this.value++;};action.run();class Local{A1 read(){return A1.this;}}if(new Local().read()!=this)throw new AssertionError();return (A1)super.clone();}}public class Main{public static void main(String[]args)throws Exception{A1 first=new A1();A1 other;synchronized(first){other=first.copy();}other.value=9;System.out.println((first!=other)+":"+first.value+":"+other.value+":"+other.base);}}`}, "true:6:9:4\n"},
		{"enumBodiesAndHelper", map[string]string{"src/main/java/receivers/Main.java": `package receivers;enum E1{A{String read(){return super.toString()+":"+((E1)this).ordinal();}},B;String read(){return super.toString()+":"+((E1)this).ordinal();}<U>E1 identity(U e1_0){return (E1)this;}}public class Main{public static void main(String[]args){System.out.println(E1.A.read()+":"+E1.B.read()+":"+(E1.A.identity("x")==E1.A));}}`}, "A:0:B:1:true\n"},
		{"throwableSuper", map[string]string{"src/main/java/receivers/Main.java": `package receivers;class A1 extends RuntimeException{String message(){return super.getMessage();}String kind(){return super.toString();}}public class Main{public static void main(String[]args){A1 value=new A1();System.out.println((value.message()==null)+":"+value.kind());}}`}, "true:receivers.A1\n"},
		{"readerFormalAndLock", map[string]string{"src/main/java/receivers/ReaderOwner.java": `package receivers;public class ReaderOwner extends java.io.Reader{public int calls;public ReaderOwner(){super();}public int read(char[] rr,int start,int length){synchronized(lock){calls++;if(length==0)return 0;rr[start]='x';return 1;}}public void close(){}}`, "src/main/java/receivers/Main.java": `package receivers;public class Main{public static void main(String[]args)throws Exception{ReaderOwner value=new ReaderOwner();System.out.println(value.read()+":"+value.calls);}}`}, "120:1\n"},
		{"inputStreamFormalAndSuper", map[string]string{"src/main/java/receivers/FilterOwner.java": `package receivers;public class FilterOwner extends java.io.FilterInputStream{public FilterOwner(){super(new java.io.ByteArrayInputStream(new byte[]{1}));}public int peek(int fr)throws java.io.IOException{return super.read()+fr;}}`, "src/main/java/receivers/Main.java": `package receivers;public class Main{public static void main(String[]args)throws Exception{System.out.println(new FilterOwner().peek(3));}}`}, "4\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.files["pom.xml"] = `<project><groupId>receiver</groupId><artifactId>ownership</artifactId><version>1</version></project>`
			runCampaignCompilerStrictProjectObservations(t, test.files, "receivers.Main", []campaignCompilerProjectObservation{
				{name: "first", stdout: test.want}, {name: "second", stdout: test.want}, {name: "third", stdout: test.want},
			})
		})
	}
}
