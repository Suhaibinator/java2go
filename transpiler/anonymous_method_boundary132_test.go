package transpiler

import "testing"

func TestAnonymousMethodBoundary132(t *testing.T) {
	for _, test := range []struct{ name, source, want string }{{name: "sourceClassMethodBoundary", source: "package members;\npublic class Main{static int trace;static int mark(int value){trace=trace*10+value;return value;}static class Base{int get(){return 0;}}public static void main(String[]args){int value=0;try{Base b=new Base(){int field=7;int get(){try{return mark(2)+field-field;}finally{mark(3);}}};value=b.get();}finally{mark(5);}System.out.println(value+\":\"+trace);}}", want: "2:235\n"}, {name: "sourceInterfaceMethodBoundary", source: "package members;\npublic class Main{static int trace;static int mark(int value){trace=trace*10+value;return value;}interface View{int get();}public static void main(String[]args){int value=0;try{View v=new View(){int field=7;public int get(){try{return mark(4)+field-field;}finally{mark(6);}}};value=v.get();}finally{mark(5);}System.out.println(value+\":\"+trace);}}", want: "4:465\n"}} {
		t.Run(test.name, func(t *testing.T) {
			runCampaignCompilerStrictProjectObservations(t, map[string]string{"src/main/java/members/Main.java": test.source, "pom.xml": "<project><groupId>members</groupId><artifactId>anonymousmethodboundary</artifactId><version>1</version></project>"}, "members.Main", []campaignCompilerProjectObservation{{name: "first", stdout: test.want}, {name: "second", stdout: test.want}, {name: "third", stdout: test.want}})
		})
	}
}
