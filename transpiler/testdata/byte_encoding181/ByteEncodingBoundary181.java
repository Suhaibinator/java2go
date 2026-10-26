// UNEXECUTED additive API reducer, not another active campaign application.
import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Arrays;

public class ByteEncodingBoundary181 {
 static String bytes(byte[] value){return Arrays.toString(value);}
 static String units(String value){String s=""+value.length();for(int i=0;i<value.length();i++)s+=","+(int)value.charAt(i);return s;}
 static String failures(int mode){
  try{
   if(mode==0)((String)null).getBytes(StandardCharsets.UTF_8);
   if(mode==1)"A".getBytes((Charset)null);
   if(mode==2)((String)null).getBytes((Charset)null);
   if(mode==3)"A".getBytes("UTF_8");
   if(mode==4)"A".getBytes((String)null);
   if(mode==5)((String)null).getBytes("UTF_8");
   return "missing";
  }catch(Exception e){return e.getClass().getSimpleName();}
 }
 public static String run() throws Exception {
  Charset[] codecs={StandardCharsets.UTF_8,StandardCharsets.US_ASCII,StandardCharsets.ISO_8859_1,StandardCharsets.UTF_16BE,StandardCharsets.UTF_16LE,StandardCharsets.UTF_16};
  char[][] cases={{},{0,'A',(char)0x7f,(char)0x80,(char)0x7ff,(char)0x800},{(char)0xe9,(char)0x65e5},{(char)0xd83d,(char)0xde00},{(char)0xd800,'x',(char)0xdc00},{(char)0xd800,(char)0xd801,(char)0xdc00,(char)0xdc01},{(char)0xfeff,(char)0xfffd}};
  StringBuilder out=new StringBuilder();
  for(Charset codec:codecs){for(char[] input:cases){
   String text=new String(input);String before=units(text);byte[] a=text.getBytes(codec),b=text.getBytes(codec);
   out.append(codec.name()).append(':').append(a!=b).append(':').append(bytes(b)).append(':');
   if(a.length>0)a[0]=(byte)(a[0]^127);
   out.append(before.equals(units(text))).append(':').append(Arrays.equals(b,text.getBytes(codec))).append('|');
  }}
  for(int mode=0;mode<6;mode++)out.append(failures(mode)).append('|');
  for(int n:new int[]{15,16,17,127,128,129,4096,65536}){
   char[] source=new char[n];for(int i=0;i<n;i++)source[i]=(char)((i%7)==0?0xd800:(i%7)==1?0xdc00:(i%7)==2?0xdc01:(i%7)==3?0x65e5:'A');
   String text=new String(source);byte[] a=text.getBytes(StandardCharsets.UTF_8),b=text.getBytes("UTF8");
   out.append(n).append(':').append(a.length).append(':').append(a!=b).append(':').append(Arrays.equals(a,b)).append(':').append(bytes(MessageDigest.getInstance("SHA-256").digest(a))).append('|');
  }
  return out.toString();
 }
}
