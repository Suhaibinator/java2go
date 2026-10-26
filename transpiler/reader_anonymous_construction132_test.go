package transpiler

import "testing"

func TestReaderAnonymousConstruction132(t *testing.T) {
	for _, test := range []struct{ name, source, want string }{
		{name: "defaultSelfBeforeFields", source: "package members;import java.io.Reader;\npublic class Main{static int trace;static boolean record(boolean value){trace++;return value;}\npublic static void main(String[]args){String text=new Reader(){public Object ReaderBase;boolean first=record(lock==this);public int read(char[]b,int o,int n){return -1;}public void close(){}String check(){return first+\":\"+(lock==this)+\":\"+(ReaderBase==null);}}.check();boolean empty=new Reader(){public int read(char[]b,int o,int n){return -1;}public void close(){}boolean check(){return lock==this;}}.check();System.out.println(text+\":\"+empty+\":\"+trace);}}", want: "true:true:true:true:1\n"},
		{name: "explicitNullAndAbruptOrder", source: "package members;import java.io.Reader;\npublic class Main{static int trace;static int fields;static Object sentinel=new Object();static RuntimeException boom=new RuntimeException();static Object argument(boolean nil,boolean fail){trace=trace*10+1;if(fail)throw boom;return nil?null:sentinel;}static boolean field(Object lock){trace=trace*10+2;fields++;return lock==sentinel;}\npublic static void main(String[]args){String ok=new Reader(argument(false,false)){public Object ReaderBase;boolean first=field(lock);public int read(char[]b,int o,int n){return -1;}public void close(){}String check(){synchronized(lock){return first+\":\"+(lock==sentinel)+\":\"+Thread.holdsLock(sentinel);}}}.check();int good=trace;trace=0;int nulls=0;try{new Reader(argument(true,false)){int initialized=++fields;public int read(char[]b,int o,int n){return -1;}public void close(){}};}catch(NullPointerException e){nulls++;}int absent=trace;trace=0;boolean exact=false;try{new Reader(argument(false,true)){int initialized=++fields;public int read(char[]b,int o,int n){return -1;}public void close(){}};}catch(RuntimeException e){exact=e==boom;}System.out.println(ok+\":\"+good+\":\"+nulls+\":\"+absent+\":\"+trace+\":\"+fields+\":\"+exact);}}", want: "true:true:true:12:1:1:1:1:true\n"},
		{name: "sourceConstructorNegative", source: "package members;\npublic class Main{static int trace;static Object sentinel=new Object();static Object argument(){trace=trace*10+1;return sentinel;}static int field(){trace=trace*10+2;return 7;}static class Reader{protected Object lock;Reader(Object value){lock=value;trace=trace*10+3;}boolean same(){return lock==sentinel;}}\npublic static void main(String[]args){Reader r=new Reader(argument()){int value=field();};System.out.println(r.same()+\":\"+trace);}}", want: "true:132\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string]string{"pom.xml": `<project><groupId>members</groupId><artifactId>anonymous</artifactId><version>1</version></project>`, "src/main/java/members/Main.java": test.source}
			runCampaignCompilerStrictProjectObservations(t, files, "members.Main", []campaignCompilerProjectObservation{{name: "first", stdout: test.want}, {name: "second", stdout: test.want}, {name: "third", stdout: test.want}})
		})
	}
}
