package transpiler

import "testing"

// These are JVM-first tests of the additive runtime API, not a claim that the
// generated String ABI has migrated. The Go programs are runtime test drivers.
func TestCampaignStringReferenceCharsetEncode(t *testing.T) {
	campaignStringCoreOperationOracle(t, "ReferenceCharsetEncode", `import java.nio.charset.*;
public class ReferenceCharsetEncode {
 public static void main(String[] args) {
  Charset[] codecs={StandardCharsets.UTF_8,StandardCharsets.US_ASCII,StandardCharsets.ISO_8859_1,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16};
  char[][] values={{},{0,'A',(char)0xe9},{(char)0xd83d,(char)0xde00},{'x',(char)0xd83d,'y',(char)0xde00},{(char)0xd800,(char)0xd801,(char)0xdc00,(char)0xdc01},{(char)0xfffd}};
  for(Charset codec:codecs){for(char[] units:values){
   String value=new String(units); byte[] first=value.getBytes(codec),second=value.getBytes(codec);
   System.out.print((first!=second)+":");
   for(byte b:first){System.out.print((int)b+",");}
   System.out.print("/");for(int i=0;i<value.length();i++){System.out.print((int)value.charAt(i)+",");}System.out.println();
  }}
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func main(){
 codecs:=[]*j.Charset{j.UTF_8,j.US_ASCII,j.ISO_8859_1,j.UTF_16BE,j.UTF_16LE,j.UTF_16}
 values:=[][]uint16{{},{0,'A',0xe9},{0xd83d,0xde00},{'x',0xd83d,'y',0xde00},{0xd800,0xd801,0xdc00,0xdc01},{0xfffd}}
 for _,codec:=range codecs{for _,units:=range values{
  value:=j.NewJavaStringUTF16(units);first,second:=j.JavaStringGetBytes(value,codec),j.JavaStringGetBytes(value,codec)
  fmt.Printf("%t:",first!=second);for _,b:=range first.Elements{fmt.Printf("%d,",b)}
  fmt.Print("/");for _,u:=range value.UTF16Copy(){fmt.Printf("%d,",u)};fmt.Println()
 }}
}`)
}

func TestCampaignStringReferenceCharsetDecode(t *testing.T) {
	campaignStringCoreOperationOracle(t, "ReferenceCharsetDecode", `import java.nio.charset.*;
public class ReferenceCharsetDecode {
 public static void main(String[] args){
  Charset[] codecs={StandardCharsets.UTF_8,StandardCharsets.US_ASCII,StandardCharsets.ISO_8859_1,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16};
  byte[][] inputs={{},{65,0,66},{-61,-87},{-16,-97,-104,-128},{-19,-96,-128},{-30,-126,65},{-30,-126},{-40,0,0,65},{0,-40,65,0},{-1,-2,65,0},{-2,-1,0,65},{-1}};
  for(Charset codec:codecs){for(byte[] data:inputs){String value=new String(data,codec),again=new String(data,codec);
   System.out.print((value!=again)+":"+value.equals(again)+":");
   for(int i=0;i<value.length();i++){System.out.print((int)value.charAt(i)+",");}System.out.println();
  }}
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func main(){
 codecs:=[]*j.Charset{j.UTF_8,j.US_ASCII,j.ISO_8859_1,j.UTF_16BE,j.UTF_16LE,j.UTF_16}
 inputs:=[][]int8{{},{65,0,66},{-61,-87},{-16,-97,-104,-128},{-19,-96,-128},{-30,-126,65},{-30,-126},{-40,0,0,65},{0,-40,65,0},{-1,-2,65,0},{-2,-1,0,65},{-1}}
 for _,codec:=range codecs{for _,data:=range inputs{
  a:=j.PrimitiveArrayLiteral(j.PrimitiveByteTypeID,data...);value,again:=j.JavaStringFromBytes(a,codec),j.JavaStringFromBytes(a,codec)
  fmt.Printf("%t:%t:",value!=again,value.Equals(again));for _,u:=range value.UTF16Copy(){fmt.Printf("%d,",u)};fmt.Println()
 }}
}`)
}

func TestCampaignStringReferenceCharsetNullAndCopy(t *testing.T) {
	campaignStringCoreOperationOracle(t, "ReferenceCharsetNullAndCopy", `import java.nio.charset.*;
public class ReferenceCharsetNullAndCopy {
 static boolean npe(Runnable action){try{action.run();return false;}catch(NullPointerException e){return true;}}
 public static void main(String[] args){
  System.out.print(npe(()->new String((byte[])null,StandardCharsets.UTF_8))+":"+npe(()->new String(new byte[0],(Charset)null))+":"+npe(()->"".getBytes((Charset)null))+":"+npe(()->((String)null).getBytes(StandardCharsets.UTF_8))+":");
  byte[] bytes={65,66};String text=new String(bytes,StandardCharsets.UTF_8);bytes[0]=90;byte[] out=text.getBytes(StandardCharsets.UTF_8);out[1]=90;
  System.out.print((int)text.charAt(0)+":"+(int)text.charAt(1)+":"+(int)text.getBytes(StandardCharsets.UTF_8)[1]);
 }
}`, `package main
import("fmt";j "github.com/NickyBoy89/java2go/stdjava")
func npe(f func())(yes bool){defer func(){yes=j.CaughtAs(recover(),"NullPointerException")}();f();return}
func main(){
 fmt.Printf("%t:%t:%t:%t:",npe(func(){j.JavaStringFromBytes(nil,j.UTF_8)}),npe(func(){j.JavaStringFromBytes(j.PrimitiveArrayLiteral[int8](j.PrimitiveByteTypeID),nil)}),npe(func(){j.JavaStringGetBytes(j.NewJavaStringUTF16(nil),nil)}),npe(func(){j.JavaStringGetBytes(nil,j.UTF_8)}))
 data:=j.PrimitiveArrayLiteral[int8](j.PrimitiveByteTypeID,65,66);text:=j.JavaStringFromBytes(data,j.UTF_8);data.Elements[0]=90;out:=j.JavaStringGetBytes(text,j.UTF_8);out.Elements[1]=90
 fmt.Printf("%d:%d:%d",text.CharAt(0),text.CharAt(1),j.JavaStringGetBytes(text,j.UTF_8).Elements[1])
}`)
}
