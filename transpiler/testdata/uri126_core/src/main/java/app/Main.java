package app;
import java.net.URI;
import java.net.URISyntaxException;
import java.io.Serializable;
import state.Cell;
import p.Owners;
public class Main {
 static String units(String text){if(text==null)return "null";String result="";for(int i=0;i<text.length();i++)result+=(i==0?"":",")+Integer.toHexString(text.charAt(i));return result;}
 static void failure(Exception failure){
  System.out.println(failure.getClass().getName()+":"+units(failure.getMessage())+":"+(failure.getCause()==null));
  if(failure instanceof URISyntaxException){URISyntaxException syntax=(URISyntaxException)failure;System.out.println(units(syntax.getInput())+":"+units(syntax.getReason())+":"+syntax.getIndex());}
 }
 static void inspect(String text){
  try {URI value=new URI(text);System.out.println("ok:"+units(value.toString()));try{System.out.println("ascii:"+units(value.toASCIIString()));}catch(Exception failure){failure(failure);}}
  catch(Exception failure){failure(failure);}
 }
 public static void main(String[] args) throws Exception {
  String mode=args.length==0?"state":args[0];
  if(mode.equals("state")){
   Cell cell=new Cell();
   cell.set("http://example.test/caf\u00e9/e\u0301?q=\u03c0#\ud83d\ude00");
   URI before=cell.value;
   System.out.println(units(cell.value.toString()));System.out.println(cell.value.toASCIIString());
   try{cell.set("http://example.test/a b");System.out.println("invalid-accepted");}catch(Exception failure){failure(failure);}finally{System.out.println("finally:"+(cell.value==before));}
   System.out.println(cell.version+":"+cell.trace);
   cell.set("urn:sample:\u00e9");System.out.println(cell.value.toASCIIString());
   cell.set("");System.out.println(units(cell.value.toString())+":"+units(cell.value.toASCIIString()));
   System.out.println(cell.version+":"+cell.trace);
  } else if(mode.equals("identity")){
   URI value=new URI("urn:sample:kept");Object object=value;Serializable serializable=value;Comparable<URI> comparable=value;
   System.out.println((object==value)+":"+(serializable==value)+":"+(comparable==value)+":"+object.toString());
   System.out.println(object.getClass().getName()+":"+(object.getClass()==URI.class));
   System.out.println(URI.class.getSuperclass().getName()+":"+Serializable.class.isInstance(value)+":"+Comparable.class.isInstance(value)+":"+URI.class.isInstance(object));
  } else if(mode.equals("namespace")){
   Owners owners=new Owners("urn:sample:\u00e9");foreign.URI source=new foreign.URI("kept");
   System.out.println(owners.observe());System.out.println(source.toASCIIString()+":"+source.toString());
   System.out.println(owners.value.getClass().getName()+":"+source.getClass().getName());
  } else if(mode.equals("exceptions")){
   inspect(null);inspect("http://x/a b");inspect("a%2");inspect("1bad:part");inspect("http://[::1");
   URISyntaxException exact=new URISyntaxException("input\ud83d\ude00","reason",5);
   try{throw exact;}catch(Exception failure){System.out.println(failure==exact);failure(failure);}finally{System.out.println("finally");}
   inspect("a\ud800");inspect("a\udc00");
  } else if(mode.equals("grammar")){
   String[] inputs={"","relative/path","../a/./b","?query","#fragment","//server/path","http://example.test","http://user@example.test:8080/a?q#f","http://[::1]:80/a","http://[2001:db8::1%25zone]/","http://host:/","http://host:bad/a","http://host:2147483648/a","http://host:-1/a","http://a_b/a","urn:sample:part","mailto:user@example.test","scheme:/path","scheme:","a:b","a%20b","a%ffb","http://x/%E2%82%AC","http://x/\u00e9","http://x/e\u0301","a\u00a0b","a\u0085b","http://[1::2::3]/","http://[::ffff:192.0.2.1]/","http://[v1.a]/","http://x/a|b","http://x/a[b]","http://x/a\\b","http://x/#a#b","//","///a","http:///a","http://","%","a%xy","\u03c0:part"};
   for(String input:inputs)inspect(input);
  } else throw new IllegalArgumentException(mode);
 }
}
