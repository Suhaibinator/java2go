package transpiler

import (
	"encoding/json"
	"fmt"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"go/token"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

const arraysUtilityProgramsJSON = `[{"name":"AdversarialOriginal","entry":"app.Main","files":{"pom.xml":"<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>bytecopy-receiver-adversarial</artifactId><version>1</version></project>\n","src/main/java/app/Main.java":"package app;\nimport ops.Operands;\npublic class Main {\n static String desc(byte[] a){if(a==null)return \"null\";String s=a.length+\":\";for(int i=0;i<a.length;i++)s=s+a[i]+\",\";return s;}\n static String failure(RuntimeException e){if(e==Operands.marker)return \"marker\";if(e instanceof NullPointerException)return \"NPE\";if(e instanceof NegativeArraySizeException)return \"NAS:\"+e.getMessage();if(e instanceof ArrayIndexOutOfBoundsException)return \"AIOOBE\";if(e instanceof IllegalArgumentException)return \"IAE:\"+e.getMessage();return \"unexpected:\"+e.getMessage();}\n static void exercise(int seed,int mode){\n  Operands.reset(seed);\n  if(mode==1)Operands.lower=null;\n  if(mode==2)Operands.upper=null;\n  if(mode==3){Operands.lower=-1;Operands.upper=Integer.MAX_VALUE;}\n  if(mode==4){Operands.lower=Integer.MIN_VALUE;Operands.upper=0;}\n  if(mode==5){Operands.lower=1;Operands.upper=0;Operands.bytes=null;}\n  if(mode==6){Operands.lower=0;Operands.upper=-1;Operands.bytes=null;}\n  if(mode==7){Operands.lower=0;Operands.upper=0;Operands.bytes=null;}\n  if(mode==8){Operands.lower=-1;Operands.upper=Integer.MAX_VALUE;Operands.bytes=null;}\n  if(mode==9){Operands.lower=5;Operands.upper=5;}\n  if(mode==10){Operands.lower=4;Operands.upper=7;}\n  try {\n   try {\n    byte[] value;\n    if(mode==11)value=calls.Explicit.mutation();\n    else if(mode==12)value=calls.Explicit.abruptTarget();\n    else if(mode==13)value=calls.Explicit.abruptArgument();\n    else value=calls.Explicit.expression();\n    System.out.println(mode+\":\"+desc(value));\n    if(value.length>0)value[0]=99;\n   }finally{Operands.finish();}\n  }catch(RuntimeException e){System.out.println(mode+\":\"+failure(e));}\n  System.out.println(Operands.trace+\":\"+Operands.cleanup+\":\"+desc(Operands.bytes));\n }\n public static void main(String[] args){\n  int seed=Integer.parseInt(args[0]);byte[] original=new byte[]{(byte)seed,-128,-1,127};\n  byte[] local=calls.Explicit.local(original);System.out.println(desc(local));System.out.println(local!=original);System.out.println(local.getClass()==byte[].class);\n  System.out.println(desc(calls.Explicit.parameter(null,original)));System.out.println(desc(calls.Explicit.fromField(original)));System.out.println(desc(calls.Wild.get(original)));\n  byte[] empty1=calls.Wild.cast(original),empty2=calls.Wild.cast(original);System.out.println(empty1!=empty2);\n  System.out.println(desc(imports.Explicit.get(original)));System.out.println(desc(imports.Wild.get(original)));\n  System.out.println(desc(lookalike.Caller.local(original)));System.out.println(desc(lookalike.Caller.expression(original)));System.out.println(lookalike.Arrays.calls+\":\"+lookalike.Arrays.trace);\n  System.out.println(desc(shadow.Shadow.hidden(original)));System.out.println(desc(shadow.Shadow.local(original)));System.out.println(desc(shadow.Shadow.parameter(new shadow.Shadow.Arrays(),original)));System.out.println(desc(shadow.Shadow.bound(new Object(),original)));\n  for(int mode=0;mode<=13;mode++)exercise(seed,mode);\n  System.out.println(desc(original));\n }\n}\n","src/main/java/calls/Explicit.java":"package calls;\nimport java.util.Arrays;import ops.Operands;\npublic class Explicit {\n public static Arrays field;\n public static byte[] local(byte[] a){Arrays receiver=null;return receiver.copyOfRange(a,0,2);}\n public static byte[] parameter(Arrays receiver,byte[] a){return receiver.copyOfRange(a,1,3);}\n public static byte[] fromField(byte[] a){return field.copyOfRange(a,0,3);}\n public static byte[] expression(){return Operands.target().copyOfRange(Operands.original(),Operands.from(),Operands.to());}\n public static byte[] mutation(){return Operands.target().copyOfRange(Operands.original(),Operands.mutate(),Operands.replace());}\n public static byte[] abruptTarget(){return Operands.abruptTarget().copyOfRange(Operands.original(),Operands.from(),Operands.to());}\n public static byte[] abruptArgument(){return Operands.target().copyOfRange(Operands.original(),Operands.abruptFrom(),Operands.to());}\n}\n","src/main/java/calls/Wild.java":"package calls;\nimport java.util.*;\npublic class Wild {\n public static byte[] get(byte[] a){Arrays receiver=null;return receiver.copyOfRange(a,0,a.length+2);}\n public static byte[] cast(byte[] a){return ((java.util.Arrays)null).copyOfRange(a,0,0);}\n}\n","src/main/java/imports/Explicit.java":"package imports;\nimport static java.util.Arrays.copyOfRange;\npublic class Explicit {public static byte[] get(byte[] a){return copyOfRange(a,1,3);}}\n","src/main/java/imports/Wild.java":"package imports;\nimport static java.util.Arrays.*;\npublic class Wild {public static byte[] get(byte[] a){return copyOfRange(a,3,6);}}\n","src/main/java/lookalike/Arrays.java":"package lookalike;\npublic class Arrays {\n public static int calls;public static int trace;\n public static Arrays target(){trace=trace*10+7;return null;}\n public static byte[] original(byte[] a){trace=trace*10+1;return a;}\n public static byte[] copyOfRange(byte[] a,int f,int t){calls++;return new byte[]{a[0],(byte)(f+70),(byte)(t+70)};}\n}\n","src/main/java/lookalike/Caller.java":"package lookalike;\npublic class Caller {\n public static byte[] local(byte[] a){Arrays receiver=null;return receiver.copyOfRange(a,0,2);}\n public static byte[] expression(byte[] a){return Arrays.target().copyOfRange(Arrays.original(a),1,3);}\n}\n","src/main/java/ops/Operands.java":"package ops;\npublic class Operands {\n public static int trace;public static int cleanup;public static byte[] bytes;\n public static Integer lower;public static Integer upper;\n public static RuntimeException marker=new IllegalStateException(\"marker\");\n public static void reset(int seed){trace=0;cleanup=0;bytes=new byte[]{(byte)seed,-128,-1,127};lower=0;upper=2;}\n public static java.util.Arrays target(){trace=trace*10+9;return null;}\n public static java.util.Arrays abruptTarget(){trace=trace*10+9;throw marker;}\n public static byte[] original(){trace=trace*10+1;return bytes;}\n public static Integer from(){trace=trace*10+2;return lower;}\n public static Integer to(){trace=trace*10+3;return upper;}\n public static Integer mutate(){trace=trace*10+2;if(bytes!=null)bytes[1]=42;return lower;}\n public static Integer replace(){trace=trace*10+3;bytes=new byte[]{8,8,8,8};return upper;}\n public static Integer abruptFrom(){trace=trace*10+2;throw marker;}\n public static void finish(){cleanup++;if(bytes!=null)bytes[3]=-7;}\n}\n","src/main/java/shadow/Shadow.java":"package shadow;\nimport static java.util.Arrays.copyOfRange;\npublic class Shadow {\n public static class Arrays {public int calls;public byte[] copyOfRange(byte[] a,int f,int t){calls++;return new byte[]{(byte)(a[0]+1),(byte)calls};}}\n static byte[] copyOfRange(byte[] a,int f,int t){return new byte[]{(byte)(a[0]+2)};}\n public static byte[] hidden(byte[] a){return copyOfRange(a,0,2);}\n public static byte[] local(byte[] a){Arrays Arrays=new Arrays();return Arrays.copyOfRange(a,0,2);}\n public static byte[] parameter(Arrays Arrays,byte[] a){return Arrays.copyOfRange(a,0,2);}\n public static<Arrays>byte[] bound(Arrays token,byte[] a){java.util.Arrays receiver=null;return receiver.copyOfRange(a,0,1);}\n}\n"},"observations":[{"seed":17,"repeat":1,"stdout":"2:17,-128,\ntrue\ntrue\n2:-128,-1,\n3:17,-128,-1,\n6:17,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:17,70,72,\n3:17,71,73,\n2:71\n1:19,\n2:18,1,\n2:18,1,\n1:17,\n0:2:17,-128,\n9123:1:4:17,-128,-1,-7,\n1:NPE\n912:1:4:17,-128,-1,-7,\n2:NPE\n9123:1:4:17,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:17,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:17,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:17,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:17,-128,-1,-7,\n11:2:17,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:17,-128,-1,-7,\n13:marker\n912:1:4:17,-128,-1,-7,\n4:17,-128,-1,127,\n","stderr":"","stdout_sha256":"c69ead087e0f43ccb78f4e61e6337ebd487c06a3aede297468b29949359e3e22","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":17,"repeat":2,"stdout":"2:17,-128,\ntrue\ntrue\n2:-128,-1,\n3:17,-128,-1,\n6:17,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:17,70,72,\n3:17,71,73,\n2:71\n1:19,\n2:18,1,\n2:18,1,\n1:17,\n0:2:17,-128,\n9123:1:4:17,-128,-1,-7,\n1:NPE\n912:1:4:17,-128,-1,-7,\n2:NPE\n9123:1:4:17,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:17,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:17,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:17,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:17,-128,-1,-7,\n11:2:17,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:17,-128,-1,-7,\n13:marker\n912:1:4:17,-128,-1,-7,\n4:17,-128,-1,127,\n","stderr":"","stdout_sha256":"c69ead087e0f43ccb78f4e61e6337ebd487c06a3aede297468b29949359e3e22","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":17,"repeat":3,"stdout":"2:17,-128,\ntrue\ntrue\n2:-128,-1,\n3:17,-128,-1,\n6:17,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:17,70,72,\n3:17,71,73,\n2:71\n1:19,\n2:18,1,\n2:18,1,\n1:17,\n0:2:17,-128,\n9123:1:4:17,-128,-1,-7,\n1:NPE\n912:1:4:17,-128,-1,-7,\n2:NPE\n9123:1:4:17,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:17,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:17,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:17,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:17,-128,-1,-7,\n11:2:17,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:17,-128,-1,-7,\n13:marker\n912:1:4:17,-128,-1,-7,\n4:17,-128,-1,127,\n","stderr":"","stdout_sha256":"c69ead087e0f43ccb78f4e61e6337ebd487c06a3aede297468b29949359e3e22","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":41,"repeat":1,"stdout":"2:41,-128,\ntrue\ntrue\n2:-128,-1,\n3:41,-128,-1,\n6:41,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:41,70,72,\n3:41,71,73,\n2:71\n1:43,\n2:42,1,\n2:42,1,\n1:41,\n0:2:41,-128,\n9123:1:4:41,-128,-1,-7,\n1:NPE\n912:1:4:41,-128,-1,-7,\n2:NPE\n9123:1:4:41,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:41,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:41,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:41,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:41,-128,-1,-7,\n11:2:41,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:41,-128,-1,-7,\n13:marker\n912:1:4:41,-128,-1,-7,\n4:41,-128,-1,127,\n","stderr":"","stdout_sha256":"6952ecfeb952d9b89be13fd957b621f556efe5b21ced546fca183efcd5691d03","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":41,"repeat":2,"stdout":"2:41,-128,\ntrue\ntrue\n2:-128,-1,\n3:41,-128,-1,\n6:41,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:41,70,72,\n3:41,71,73,\n2:71\n1:43,\n2:42,1,\n2:42,1,\n1:41,\n0:2:41,-128,\n9123:1:4:41,-128,-1,-7,\n1:NPE\n912:1:4:41,-128,-1,-7,\n2:NPE\n9123:1:4:41,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:41,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:41,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:41,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:41,-128,-1,-7,\n11:2:41,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:41,-128,-1,-7,\n13:marker\n912:1:4:41,-128,-1,-7,\n4:41,-128,-1,127,\n","stderr":"","stdout_sha256":"6952ecfeb952d9b89be13fd957b621f556efe5b21ced546fca183efcd5691d03","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":41,"repeat":3,"stdout":"2:41,-128,\ntrue\ntrue\n2:-128,-1,\n3:41,-128,-1,\n6:41,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:41,70,72,\n3:41,71,73,\n2:71\n1:43,\n2:42,1,\n2:42,1,\n1:41,\n0:2:41,-128,\n9123:1:4:41,-128,-1,-7,\n1:NPE\n912:1:4:41,-128,-1,-7,\n2:NPE\n9123:1:4:41,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:41,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:41,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:41,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:41,-128,-1,-7,\n11:2:41,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:41,-128,-1,-7,\n13:marker\n912:1:4:41,-128,-1,-7,\n4:41,-128,-1,127,\n","stderr":"","stdout_sha256":"6952ecfeb952d9b89be13fd957b621f556efe5b21ced546fca183efcd5691d03","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":97,"repeat":1,"stdout":"2:97,-128,\ntrue\ntrue\n2:-128,-1,\n3:97,-128,-1,\n6:97,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:97,70,72,\n3:97,71,73,\n2:71\n1:99,\n2:98,1,\n2:98,1,\n1:97,\n0:2:97,-128,\n9123:1:4:97,-128,-1,-7,\n1:NPE\n912:1:4:97,-128,-1,-7,\n2:NPE\n9123:1:4:97,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:97,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:97,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:97,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:97,-128,-1,-7,\n11:2:97,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:97,-128,-1,-7,\n13:marker\n912:1:4:97,-128,-1,-7,\n4:97,-128,-1,127,\n","stderr":"","stdout_sha256":"b97471ba45403ce5cc3f48532d2a480c364f5915edb4693efc14dda5a355611e","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":97,"repeat":2,"stdout":"2:97,-128,\ntrue\ntrue\n2:-128,-1,\n3:97,-128,-1,\n6:97,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:97,70,72,\n3:97,71,73,\n2:71\n1:99,\n2:98,1,\n2:98,1,\n1:97,\n0:2:97,-128,\n9123:1:4:97,-128,-1,-7,\n1:NPE\n912:1:4:97,-128,-1,-7,\n2:NPE\n9123:1:4:97,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:97,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:97,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:97,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:97,-128,-1,-7,\n11:2:97,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:97,-128,-1,-7,\n13:marker\n912:1:4:97,-128,-1,-7,\n4:97,-128,-1,127,\n","stderr":"","stdout_sha256":"b97471ba45403ce5cc3f48532d2a480c364f5915edb4693efc14dda5a355611e","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},{"seed":97,"repeat":3,"stdout":"2:97,-128,\ntrue\ntrue\n2:-128,-1,\n3:97,-128,-1,\n6:97,-128,-1,127,0,0,\ntrue\n2:-128,-1,\n3:127,0,0,\n3:97,70,72,\n3:97,71,73,\n2:71\n1:99,\n2:98,1,\n2:98,1,\n1:97,\n0:2:97,-128,\n9123:1:4:97,-128,-1,-7,\n1:NPE\n912:1:4:97,-128,-1,-7,\n2:NPE\n9123:1:4:97,-128,-1,-7,\n3:NAS:-2147483648\n9123:1:4:97,-128,-1,-7,\n4:NAS:-2147483648\n9123:1:4:97,-128,-1,-7,\n5:IAE:1 > 0\n9123:1:null\n6:NPE\n9123:1:null\n7:NPE\n9123:1:null\n8:NAS:-2147483648\n9123:1:null\n9:AIOOBE\n9123:1:4:97,-128,-1,-7,\n10:3:0,0,0,\n9123:1:4:97,-128,-1,-7,\n11:2:97,42,\n9123:1:4:8,8,8,-7,\n12:marker\n9:1:4:97,-128,-1,-7,\n13:marker\n912:1:4:97,-128,-1,-7,\n4:97,-128,-1,127,\n","stderr":"","stdout_sha256":"b97471ba45403ce5cc3f48532d2a480c364f5915edb4693efc14dda5a355611e","stderr_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}]},{"name":"declarationOriginShadow","files":{"src/main/java/provider/Factory.java":"package provider; import java.util.Arrays; public class Factory {public static Arrays field;public static int trace;public static Arrays utility(){trace=trace*10+1;return null;}public static byte[] data(){trace=trace*10+2;return new byte[]{6,7,8};}public static Integer start(){trace=trace*10+3;return 1;}public static Integer end(){trace=trace*10+4;return 3;}}\n","src/main/java/app/Main.java":"package app;import provider.Factory; public class Main {static class java {static class util {static class Arrays {static byte[] copyOfRange(byte[] a,int f,int t){return new byte[]{91};}}}}public static void main(String[] args){System.out.println(Factory.utility().copyOfRange(Factory.data(),Factory.start(),Factory.end())[0]);System.out.println(Factory.trace);System.out.println(Factory.field.copyOfRange(new byte[]{9},0,1)[0]);System.out.println(java.util.Arrays.copyOfRange(new byte[]{1},0,1)[0]);java.util.Arrays source=null;System.out.println(source.copyOfRange(new byte[]{2},0,1)[0]);}}\n","pom.xml":"<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>utility-reference</artifactId><version>1</version></project>\n"},"entry":"app.Main","source_manifest":{"src/main/java/provider/Factory.java":"e0a75e7e4e85220cac4b9ffbf84ad894b8c04d28494922de89ff5ac1d891406d","src/main/java/app/Main.java":"e0faf0570ed56e55de7046a49808950297aeda446d291290320d917a461c1bba","pom.xml":"b47cceb6831c2710714cc5420c8e5dce45a1b9efe39d50f350749319b94ddf87"},"javac_exit":0,"observations":[{"repeat":1,"stdout":"7\n1234\n9\n91\n91\n","stderr":""},{"repeat":2,"stdout":"7\n1234\n9\n91\n91\n","stderr":""},{"repeat":3,"stdout":"7\n1234\n9\n91\n91\n","stderr":""}]},{"name":"canonicalReferences","files":{"src/main/java/explicit/Holder.java":"package explicit;import java.util.Arrays;public class Holder {public static Arrays field;public static Arrays give(){return field;}public static byte[] use(Arrays value,byte[] data){return value.copyOfRange(data,0,2);}}\n","src/main/java/wild/Holder.java":"package wild;import java.util.*;public class Holder {public static Arrays field;public static byte[] use(Arrays value,byte[] data){return value.copyOfRange(data,0,2);}}\n","src/main/java/app/Main.java":"package app;public class Main {static <Arrays> byte[] generic(Arrays token,byte[] data){java.util.Arrays canonical=null;return canonical.copyOfRange(data,0,2);} public static void main(String[] args){byte[] a={4,5};System.out.println(explicit.Holder.give()==null);Object reference=explicit.Holder.field;System.out.println(reference==null);java.util.Arrays cast=(java.util.Arrays)reference;System.out.println(cast==null);System.out.println(explicit.Holder.use(cast,a)[1]);System.out.println(wild.Holder.use(null,a)[0]);System.out.println(generic(new Object(),a)[1]);System.out.println(((java.util.Arrays)null).copyOfRange(a,0,0).length);}}\n","pom.xml":"<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>utility-reference</artifactId><version>1</version></project>\n"},"entry":"app.Main","source_manifest":{"src/main/java/explicit/Holder.java":"7a5b8b763f35279c4e838616d6cee12b2302ab416f103f2e11ea42ca981e044c","src/main/java/wild/Holder.java":"4417c1e6a8439598f5ffbd8b61036628f44b5a5acfb2ea10f4d5f32b9c8d58e3","src/main/java/app/Main.java":"737a66dce3fb499fc6bf67f20a62ed74f1c2e8938e49c71a211f63e8f2cef5a2","pom.xml":"b47cceb6831c2710714cc5420c8e5dce45a1b9efe39d50f350749319b94ddf87"},"javac_exit":0,"observations":[{"repeat":1,"stdout":"true\ntrue\ntrue\n5\n4\n5\n0\n","stderr":""},{"repeat":2,"stdout":"true\ntrue\ntrue\n5\n4\n5\n0\n","stderr":""},{"repeat":3,"stdout":"true\ntrue\ntrue\n5\n4\n5\n0\n","stderr":""}]},{"name":"foreignSourceBinders","files":{"src/main/java/foreign/Arrays.java":"package foreign;public class Arrays {public static int trace;public static Arrays give(){trace=trace*10+1;return null;}public static byte[] data(){trace=trace*10+2;return new byte[]{3};}public static byte[] copyOfRange(byte[] a,int f,int t){trace=trace*10+3;return new byte[]{(byte)(a[0]+70)};}}\n","src/main/java/app/Main.java":"package app;public class Main {static class Arrays {byte[] copyOfRange(byte[] a,int f,int t){return new byte[]{81};}}static <Arrays extends foreign.Arrays> byte[] bound(Arrays receiver,byte[] a){return receiver.copyOfRange(a,0,1);}static byte[] value(Arrays Arrays,byte[] a){return Arrays.copyOfRange(a,0,1);}public static void main(String[] args){foreign.Arrays receiver=null;System.out.println(receiver.copyOfRange(new byte[]{4},0,1)[0]);System.out.println(foreign.Arrays.give().copyOfRange(foreign.Arrays.data(),0,1)[0]);System.out.println(foreign.Arrays.trace);System.out.println(bound(receiver,new byte[]{5})[0]);System.out.println(value(new Arrays(),new byte[]{6})[0]);}}\n","pom.xml":"<project xmlns=\"http://maven.apache.org/POM/4.0.0\"><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>utility-reference</artifactId><version>1</version></project>\n"},"entry":"app.Main","source_manifest":{"src/main/java/foreign/Arrays.java":"35f012499aca0d4ce55b5e2e01e83b78e4c1b3ceacade8693328181ec116cda3","src/main/java/app/Main.java":"94c18914ebe5ee3b99bfdecb01420371bf70aeb0d7b8a682e171af123919cf4e","pom.xml":"b47cceb6831c2710714cc5420c8e5dce45a1b9efe39d50f350749319b94ddf87"},"javac_exit":0,"observations":[{"repeat":1,"stdout":"74\n73\n3123\n75\n81\n","stderr":""},{"repeat":2,"stdout":"74\n73\n3123\n75\n81\n","stderr":""},{"repeat":3,"stdout":"74\n73\n3123\n75\n81\n","stderr":""}]}]`

func arraysUtilityFullProject(t *testing.T, name string) {
	t.Helper()
	var programs []struct {
		Name         string
		Entry        string
		Files        map[string]string
		Observations []struct {
			Seed   int
			Repeat int
			Stdout string
			Stderr string
		}
	}
	if err := json.Unmarshal([]byte(arraysUtilityProgramsJSON), &programs); err != nil {
		t.Fatal(err)
	}
	for _, p := range programs {
		if p.Name == name {
			var observations []campaignCompilerProjectObservation
			for _, o := range p.Observations {
				var args []string
				if name == "AdversarialOriginal" {
					args = []string{strconv.Itoa(o.Seed)}
				}
				observations = append(observations, campaignCompilerProjectObservation{name: fmt.Sprintf("seed-%d-repeat-%d", o.Seed, o.Repeat), args: args, stdout: o.Stdout, stderr: o.Stderr})
			}
			runCampaignCompilerStrictProjectObservations(t, p.Files, p.Entry, observations)
			return
		}
	}
	t.Fatal("unknown full program")
}
func TestArraysUtilityAdversarialOriginalJDK21(t *testing.T) {
	arraysUtilityFullProject(t, "AdversarialOriginal")
}
func TestArraysUtilityDeclarationOriginShadowJDK21(t *testing.T) {
	arraysUtilityFullProject(t, "declarationOriginShadow")
}
func TestArraysUtilityCanonicalReferencesJDK21(t *testing.T) {
	arraysUtilityFullProject(t, "canonicalReferences")
}
func TestArraysUtilityForeignSourceBindersJDK21(t *testing.T) {
	arraysUtilityFullProject(t, "foreignSourceBinders")
}
func TestArraysUtilitySharedSelection(t *testing.T) {
	cases := []struct {
		name, source string
		selected     bool
	}{
		{"parameter", `import java.util.Arrays;class Main{static byte[] run(Arrays utility,byte[] a,Integer f){return utility.copyOfRange(a,f,2);}}`, true},
		{"qualifiedbinder", `class Main{static <Arrays> byte[] run(Arrays token,byte[] a){java.util.Arrays utility=null;return utility.copyOfRange(a,0,2);}}`, true},
		{"cast", `class Main{static byte[] run(byte[] a){return ((java.util.Arrays)null).copyOfRange(a,0,0);}}`, true},
		{"source", `class Main{static class Arrays{static byte[] copyOfRange(byte[] a,int f,int t){return a;}}static byte[] run(Arrays value,byte[] a){return value.copyOfRange(a,0,1);}}`, false},
		{"foreign", `import foreign.Arrays;class Main{static byte[] run(Arrays value,byte[] a){return value.copyOfRange(a,0,1);}}`, false},
		{"unanchored", `class Main{static byte[] run(Arrays value,byte[] a){return value.copyOfRange(a,0,1);}}`, false},
		{"char", `import java.util.Arrays;class Main{static char[] run(Arrays value,char[] a){return value.copyOfRange(a,0,1);}}`, false},
		{"long", `import java.util.Arrays;class Main{static byte[] run(Arrays value,byte[] a,long f){return value.copyOfRange(a,f,1);}}`, false},
		{"binder", `class Main<Arrays>{byte[] run(Arrays value,byte[] a){return value.copyOfRange(a,0,1);}}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := setupParseHelper(t, c.source)
			ctx := h.Ctx.Clone()
			ctx.className = ctx.currentClass.Class.Name
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
			node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if got := arraysByteCopyRangeSelected(node, ctx, h.File.Source); got != c.selected {
				t.Fatalf("selected=%v want=%v", got, c.selected)
			}
			expected := staticIntrinsicExpectedArguments[intrinsicKey{"Arrays", "copyOfRange"}](node, ctx, h.File.Source)
			result, known := instanceIntrinsicDerivedResultTypes[intrinsicKey{"Arrays", "copyOfRange"}](node, ctx, h.File.Source)
			if c.selected {
				if !slices.Equal(expected, []string{"byte[]", "int", "int"}) || !known || result != "byte[]" {
					t.Fatalf("expected %v result %s/%v", expected, result, known)
				}
			} else if expected != nil || known {
				t.Fatalf("declined selection acquired ABI %v %s/%v", expected, result, known)
			}
		})
	}
}
func TestArraysUtilityStaticStageCoreEquivalence(t *testing.T) {
	qualifier := ast.NewIdent("alreadyEvaluatedQualifier")
	call := ast.NewIdent("alreadyConvertedCall")
	for _, results := range []*ast.FieldList{nil, {List: []*ast.Field{{Type: ast.NewIdent("int32")}}}} {
		expected := &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{qualifier}}, invocationClosureCallStatement(call, results)}}}}
		got := stageStaticQualifierWithResults(qualifier, call, results)
		if !reflect.DeepEqual(got, expected) {
			t.Fatal("static staging differs from frozen source IIFE")
		}
	}
}

func TestArraysUtilityRetainsDeclarationContext(t *testing.T) {
	h := setupParseHelper(t, `import java.util.Arrays;class Owner{static Arrays field;}`)
	actual := declaredJavaTypeOrigin(symbol.JavaType{Original: "Arrays"}, h.Ctx)
	if actual.declaringOwner != h.Ctx.currentClass {
		t.Fatal("missing declaration owner")
	}
	caller := h.Ctx.Clone()
	caller.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("Arrays", nil)}
	if !arraysUtilityOriginCanonical("caller.requalified.Arrays", actual, caller) {
		t.Fatal("borrowed caller spelling or binder")
	}
	detached := actual
	detached.declaringOwner = &symbol.ClassScope{Class: &symbol.Definition{Name: "Detached", OriginalName: "Detached"}}
	if arraysUtilityOriginCanonical("Arrays", detached, caller) {
		t.Fatal("borrowed caller file for detached owner")
	}
	parameter := symbol.NewTypeParam("T", nil)
	formal := inferredJavaTypeOrigin{parameter: parameter.Declaration, javaType: "T"}
	substituted := substituteJavaTypeOrigin(formal, map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{parameter.Declaration: actual})
	if substituted.declaringOwner != actual.declaringOwner || substituted.javaType != actual.javaType || substituted.parameter != nil {
		t.Fatal("method type substitution lost argument provenance")
	}
	if arraysUtilityOriginCanonical("Arrays", formal, caller) {
		t.Fatal("binder admitted")
	}
	unresolved := actual
	unresolved.unresolved = true
	if arraysUtilityOriginCanonical("Arrays", unresolved, caller) {
		t.Fatal("unresolved origin admitted")
	}
	source := actual
	source.nominalScope = h.Ctx.currentClass
	if arraysUtilityOriginCanonical("Arrays", source, caller) {
		t.Fatal("source nominal admitted")
	}
}
