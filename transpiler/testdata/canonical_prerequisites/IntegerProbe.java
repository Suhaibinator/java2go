package prereq;
public class IntegerProbe {
 static int calls=0;
 static int argument(int value){calls++;return value;}
 static <T extends String> int length(T text){return text.length();}
 public static void main(String[] args){
  int[] values={Integer.MIN_VALUE,-17,-1,0,1,17,Integer.MAX_VALUE};
  for(int value:values){String a=Integer.toString(argument(value));String b=Integer.toString(value);System.out.println(a+":"+a.length()+":"+length(a)+":"+(a!=b)+":"+Integer.parseInt(a)+":"+(int)a.charAt(0));}
  System.out.println("calls:"+calls);
  try{Integer.parseInt(new String(new char[]{'1',(char)0xd800,'2'}));}catch(NumberFormatException e){String s=e.getMessage();String units="";for(int i=0;i<s.length();i++){if(i>0)units+=",";units+=(int)s.charAt(i);}System.out.println("parse:"+units+":"+(e.getCause()==null));}
 }
}
