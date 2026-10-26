public class SearchProbe {
 static final Object lock=new Object();
 static final RuntimeException marker=new RuntimeException("search-marker");
 static java.lang.String trace="";
 static boolean abruptSecond=false;
 static java.lang.String receiver(java.lang.String value){trace+="r";return value;}
 static java.lang.String needle(java.lang.String value){trace+="n";return value;}
 static <T> T argument(java.lang.String id,T value){trace+=id;if(abruptSecond&&id.equals("f"))throw marker;return value;}
 static <T extends java.lang.Integer> int bound(java.lang.String text,T point,T from,boolean backwards){
  return backwards?receiver(text).lastIndexOf(argument("n",point),argument("f",from)):receiver(text).indexOf(argument("n",point),argument("f",from));
 }
 static <T extends java.lang.Character> int charBound(java.lang.String text,T point){return receiver(text).indexOf(argument("n",point));}
 static class String {
  int indexOf(int value){return 700+value;}
  int lastIndexOf(java.lang.String value,int from){return 800+from+value.length();}
 }
 static void controlled(java.lang.String id,java.lang.String text,java.lang.String value,java.lang.Integer point,java.lang.Integer from,java.lang.Integer end,int mode){
  trace="";
  try {
   int result;
   if(mode==0)result=receiver(text).indexOf(needle(value),argument("f",from));
   else if(mode==1)result=receiver(text).lastIndexOf(needle(value),argument("f",from));
   else if(mode==2)result=receiver(text).indexOf(needle(value),argument("f",from),argument("e",end));
   else if(mode==3)result=receiver(text).indexOf(argument("n",point),argument("f",from));
   else if(mode==4)result=receiver(text).lastIndexOf(argument("n",point),argument("f",from));
   else result=receiver(text).indexOf(argument("n",point),argument("f",from),argument("e",end));
   System.out.println(id+":ok:"+result+":"+trace+":"+Thread.holdsLock(lock));
  }catch(Throwable failure){
   java.lang.String message=failure instanceof StringIndexOutOfBoundsException?failure.getMessage():"";
   System.out.println(id+":"+failure.getClass().getName()+":"+(failure==marker)+":"+trace+":"+Thread.holdsLock(lock)+":"+message);
  }
 }
 static void generic(java.lang.String id,java.lang.String text,java.lang.Integer point,java.lang.Integer from,boolean back){
  trace="";
  try{System.out.println(id+":ok:"+bound(text,point,from,back)+":"+trace+":"+Thread.holdsLock(lock));}
  catch(Throwable failure){System.out.println(id+":"+failure.getClass().getName()+":"+(failure==marker)+":"+trace+":"+Thread.holdsLock(lock));}
 }
 public static void main(java.lang.String[] args){
  int seed=java.lang.Integer.parseInt(args[0]);char latin=(char)('a'+seed%26);
  java.lang.String[] texts={"A😀|møøse|😀Z","A😀Z😀",new java.lang.String(new char[]{latin,0,(char)0xd800,'x',(char)0xdc00,latin,(char)0xd83d,(char)0xde00,'z'})};
  int[] points={-1,0,latin,'|',0x1f600,0xd83d,0xde00,0xd800,0xdc00,0x10000,0x10ffff,0x110000};
  java.lang.String[] words={"","😀",new java.lang.String(new char[]{(char)0xd800}),new java.lang.String(new char[]{(char)0xdc00}),new java.lang.String(new char[]{latin}),"Z","møøse","missing"};
  System.out.println("seed:"+seed);
  System.out.println("historical:"+CampaignStringSearch.run());
  for(int t=0;t<texts.length;t++){
   java.lang.String text=texts[t];int length=text.length();
   int[] starts={java.lang.Integer.MIN_VALUE,-3,-1,0,1,2,3,length-1,length,length+1,java.lang.Integer.MAX_VALUE};
   for(int point:points){
    System.out.println("point:"+t+":"+point+":"+text.indexOf(point)+":"+text.lastIndexOf(point));
    for(int from:starts)System.out.println("point-from:"+t+":"+point+":"+from+":"+text.indexOf(point,from)+":"+text.lastIndexOf(point,from));
   }
   for(int w=0;w<words.length;w++){
    java.lang.String word=words[w];System.out.println("word:"+t+":"+w+":"+text.indexOf(word)+":"+text.lastIndexOf(word));
    for(int from:starts)System.out.println("word-from:"+t+":"+w+":"+from+":"+text.indexOf(word,from)+":"+text.lastIndexOf(word,from));
   }
   int[][] ranges={{0,length},{1,length},{2,3},{length,length},{0,0},{1,1},{0,length-1}};
   for(int[] range:ranges){
    for(int point:new int[]{latin,0x1f600,0xd83d,0xde00,-1,0x110000})System.out.println("point-range:"+t+":"+point+":"+range[0]+":"+range[1]+":"+text.indexOf(point,range[0],range[1]));
    for(int w:new int[]{0,1,2,3,6,7})System.out.println("word-range:"+t+":"+w+":"+range[0]+":"+range[1]+":"+text.indexOf(words[w],range[0],range[1]));
   }
  }
  synchronized(lock){
   java.lang.String text=texts[0];int length=text.length();
   for(int mode=0;mode<6;mode++){
    controlled("normal-"+mode,text,"😀",0x1f600,1,length,mode);
    controlled("receiver-null-"+mode,null,"😀",0x1f600,1,length,mode);
    controlled("from-null-"+mode,text,"😀",0x1f600,null,length,mode);
    controlled("from-null-receiver-null-"+mode,null,"😀",0x1f600,null,length,mode);
    if(mode>=3)controlled("point-null-"+mode,text,"😀",null,1,length,mode);
    else controlled("needle-null-"+mode,text,null,0x1f600,1,length,mode);
    abruptSecond=true;
    controlled("second-abrupt-"+mode,text,"😀",0x1f600,1,length,mode);
    if(mode>=3)controlled("point-null-second-abrupt-"+mode,text,"😀",null,1,length,mode);
    else controlled("needle-null-second-abrupt-"+mode,text,null,0x1f600,1,length,mode);
    abruptSecond=false;
   }
   for(int mode:new int[]{2,5}){
    controlled("negative-range-"+mode,text,"😀",0x1f600,-1,length,mode);
    controlled("reversed-range-"+mode,text,"",-1,2,1,mode);
    controlled("large-range-"+mode,text,"møøse",0x110000,0,length+1,mode);
    controlled("end-null-"+mode,text,"😀",0x1f600,0,null,mode);
    controlled("null-needle-invalid-range-"+mode,text,null,null,-1,length+1,mode);
   }
   generic("bound",text,0x1f600,2,false);generic("bound-last",text,0x1f600,java.lang.Integer.MAX_VALUE,true);
   generic("bound-null",text,null,1,false);abruptSecond=true;generic("bound-null-second-abrupt",text,null,1,false);abruptSecond=false;
   trace="";System.out.println("bound-char:"+charBound(text,'|')+":"+trace+":"+Thread.holdsLock(lock));
   String shadow=new String();System.out.println("source-owner:"+shadow.indexOf(4)+":"+shadow.lastIndexOf("x",3));
   foreign.String foreign=new foreign.String();System.out.println("foreign-owner:"+foreign.indexOf("😀",2)+":"+foreign.lastIndexOf(7));
  }
 }
}
