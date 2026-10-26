public final class ArrayNullABI192 {
 static final class Node<T> { }
 static final class Cell<T> { T value; T read(){return value;} }
 static int[] field;
 static int effects;
 static int[] produce(){effects++;return null;}
 static boolean pObject(Object[] a){return a==null;}
 static boolean pPrimitive(int[] a){return a==null;}
 static boolean pPrimitiveReverse(byte[] a){return null!=a;}
 static boolean pNestedPrimitive(int[][] a){return a==null;}
 static boolean pNestedReference(Node<?>[][] a){return a!=null;}
 static boolean pGenericReference(Node<String>[] a){return a==null;}
 static boolean pNullableAlias(Node<String>[] original){Node<String>[] nullableAlias=original;return nullableAlias==null;}
 static <T> boolean pGenericArray(T[] a){return a==null;}
 static boolean pParenthesized(String[] a){return ((a))!=null;}
 static boolean pReverseReference(Object[] a){return null==a;}
 static boolean nObject(Object a){return a==null;}
 static <T> boolean nScalar(T a){return a==null;}
 static boolean nWildcard(Cell<?> a){return a.read()==null;}
 static boolean nErasedField(Cell<int[]> a){return a.value==null;}
 static boolean nCall(){return produce()==null;}
 static boolean nExplicitCast(Object a){return ((int[])a)==null;}
 static boolean nField(){return field==null;}
 static boolean nString(String a){return a==null;}
 static boolean nInterface(Runnable a){return a==null;}
 static boolean nArrayElement(Object[][] a){return a[0]==null;}
 public static void main(String[]args){
  int seed=Integer.parseInt(args[0]);
  Node<String>[] absent=null;
  int[] primitive=(seed%2==0)?null:new int[0];
  Object[] objects=(seed%2==0)?new Object[0]:null;
  Cell<int[]> cell=new Cell<>();
  Object erased=(int[])null;
  System.out.println(seed+":"+pObject(objects)+":"+pPrimitive(primitive)+":"+pPrimitiveReverse(new byte[0]));
  System.out.println(pNestedPrimitive(null)+":"+pNestedReference(new Node<?>[0][])+":"+pGenericReference(absent)+":"+pNullableAlias(absent));
  System.out.println(pGenericArray(objects)+":"+pParenthesized(args)+":"+pReverseReference(objects));
  System.out.println(nObject(erased)+":"+nScalar(erased)+":"+nWildcard(cell)+":"+nErasedField(cell));
  System.out.println(nExplicitCast(erased)+":"+nField()+":"+nString(null)+":"+nInterface(null)+":"+nArrayElement(new Object[][]{null}));
  Cell raw=cell;raw.value="polluted";
  System.out.println("polluted:"+nErasedField(cell)+":"+nObject(cell.value));
  boolean first=pPrimitive(produce());
  boolean skipped=false&&pPrimitive(produce());
  boolean called=nCall();
  System.out.println("effects:"+first+":"+skipped+":"+called+":"+effects);
  try {Cell<int[]> missing=null;System.out.println(missing.value==null);}catch(NullPointerException expected){System.out.println("null-receiver");}
 }
}
