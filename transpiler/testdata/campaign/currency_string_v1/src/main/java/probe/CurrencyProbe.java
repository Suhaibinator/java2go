package probe;
import java.util.Currency;
import java.io.Serializable;
import java.util.concurrent.CountDownLatch;
public class CurrencyProbe {
 static Currency CURRENCY=Currency.getInstance("USD");
 static String trace="";
 static String argument(String value){trace+="a";return value;}
 static Currency qualifier(){trace+="q";return null;}
 static String throwing(){trace+="t";throw new IllegalStateException("argument");}
 static void inspect(String label,String value){try{Currency c=Currency.getInstance(value);System.out.println(label+":"+c.getCurrencyCode()+":"+(c.toString()==c.getCurrencyCode()));}catch(IllegalArgumentException e){System.out.println(label+":IllegalArgumentException");}catch(NullPointerException e){System.out.println(label+":NullPointerException");}}
 static String accept(Serializable value){return value.getClass().getName();}
 public static void main(String[] args)throws Exception{
  int number=0;for(String value:new String[]{"USD","JPY","EUR","XXX","XAU","ADP","ANG","XCG","ZWG","ZZZ","usd","US","USDD","U\u0000D","\u00dcSD","US\ud800","\ud83d\ude00D"})inspect("case"+(number++),value);
  inspect("null",(String)null);
  String firstCode=new String("CHF"),secondCode=new String("CHF");Currency first=Currency.getInstance(firstCode),second=Currency.getInstance(secondCode);
  System.out.println("cache:"+(first==second)+":"+(first.getCurrencyCode()==firstCode)+":"+(first.getCurrencyCode()==secondCode)+":"+(first.toString()==firstCode));
  Object raw=first;Serializable marker=first;
  System.out.println("nominal:"+(raw instanceof Currency)+":"+(raw instanceof Serializable)+":"+(first.getClass()==Currency.class)+":"+Currency.class.isInstance(raw)+":"+(first==(Currency)marker)+":"+accept(first));
  System.out.println("object:"+raw.toString()+":"+raw.equals(second)+":"+(raw.hashCode()==second.hashCode())+":"+CURRENCY.getCurrencyCode());
  trace="";try{qualifier().getInstance(argument(null));}catch(NullPointerException e){System.out.println("order:"+trace+":"+e.getClass().getSimpleName());}
  trace="";try{qualifier().getInstance(throwing());}catch(IllegalStateException e){System.out.println("throw:"+trace+":"+e.getMessage());}
  trace="";System.out.println("static:"+qualifier().getInstance(argument("GBP"))+":"+trace);
  try{((Currency)null).getCurrencyCode();}catch(NullPointerException e){System.out.println("null-get");}
  try{((Currency)null).toString();}catch(NullPointerException e){System.out.println("null-string");}
  CountDownLatch start=new CountDownLatch(1);Currency[] results=new Currency[4];String[] texts={new String("NOK"),new String("NOK"),new String("NOK"),new String("NOK")};Thread[] threads=new Thread[4];
  for(int k=0;k<4;k++){final int index=k;threads[k]=new Thread(()->{try{start.await();results[index]=Currency.getInstance(texts[index]);}catch(InterruptedException e){throw new RuntimeException(e);}});threads[k].start();}
  start.countDown();for(Thread t:threads)t.join();boolean same=true,winning=false;for(int k=0;k<4;k++){same&=results[k]==results[0];winning|=results[0].getCurrencyCode()==texts[k];}System.out.println("publish:"+same+":"+winning+":"+results[0]);
 }
}
