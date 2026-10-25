package transpiler

import "testing"

func TestInetAddressStrictJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>inet-address</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;import java.net.InetAddress;import java.net.UnknownHostException;
 public class Main {
 public static void main(String[]args)throws Exception{
  InetAddress a=InetAddress.getByName("2001:db8::1");InetAddress b=InetAddress.getByName("[2001:db8::1]");Object erased=a;
  System.out.println(a.getHostAddress()+":"+a.equals(b)+":"+(a==b)+":"+a.hashCode()+":"+a.toString()+":"+a.getClass().getName()+":"+erased.equals(b));
  InetAddress v4=InetAddress.getByName("::ffff:192.0.2.1");System.out.println(v4.getHostAddress()+":"+v4.getClass().getName());
  InetAddress loop=InetAddress.getByName(null);System.out.println(loop.toString()+":"+(loop==InetAddress.getByName("")));
  try{InetAddress.getByName("[::g]");System.out.println("bad");}catch(UnknownHostException e){System.out.println(e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null));}
  try{InetAddress.getByName("0x7f000001");System.out.println("bad");}catch(java.io.IOException e){System.out.println(e.getMessage()+":"+e.getCause().getClass().getName()+":"+e.getCause().getMessage());}
  String text=new String(new char[]{'m',(char)0xd800});UnknownHostException e=new UnknownHostException(text);System.out.println((e.getMessage()==text)+":"+(e.getCause()==null)+":"+UnknownHostException.class.getSuperclass().getName());
 }
 }`}, "probe.Main", "2001:db8:0:0:0:0:0:1:true:false:536939705:/2001:db8:0:0:0:0:0:1:java.net.Inet6Address:true\n192.0.2.1:java.net.Inet4Address\nlocalhost/127.0.0.1:true\njava.net.UnknownHostException:[::g]: invalid IPv6 address literal:true\n0x7f000001:java.lang.IllegalArgumentException:Invalid IP address literal: 0x7f000001\ntrue:true:java.io.IOException\n")
}
