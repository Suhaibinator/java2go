import java.util.concurrent.atomic.AtomicIntegerFieldUpdater;
public class AtomicUpdaterArgumentOrderProbe {
 volatile int count;
 static int trace;
 static int abrupt;
 static AtomicIntegerFieldUpdater<AtomicUpdaterArgumentOrderProbe> receiver(){trace=trace*10+1;return null;}
 static AtomicUpdaterArgumentOrderProbe target(){trace=trace*10+2;if(abrupt==1)throw new IllegalArgumentException();return null;}
 static int value(){trace=trace*10+3;if(abrupt==2)throw new IllegalStateException();return 8;}
 static int targetJava2goExecution(){return -91;}
 static int valueJava2goExecution(){return -92;}
 public static void main(String[] args){
  abrupt=0;trace=0;
  try{receiver().set(target(),value());System.out.println("accepted");}catch(NullPointerException e){System.out.println("null:"+trace);}
  abrupt=1;trace=0;
  try{receiver().set(target(),value());System.out.println("accepted");}catch(IllegalArgumentException e){System.out.println("target:"+trace);}
  abrupt=2;trace=0;
  try{receiver().set(target(),value());System.out.println("accepted");}catch(IllegalStateException e){System.out.println("value:"+trace);}
 }
}
