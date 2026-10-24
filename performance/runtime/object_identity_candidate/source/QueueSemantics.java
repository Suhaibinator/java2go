interface QueueTag { int value(); }
class QueueBase { public int value(){return 3;} }
class QueueMiddle extends QueueBase {}
class QueueLeaf extends QueueMiddle implements QueueTag { @Override public int value(){return 7;} }
class QueueOther {}
public final class QueueSemantics {
 public static int check(){
  int result=0;
  QueueLeaf leaf=new QueueLeaf();
  Object[] objects=new QueueBase[]{leaf};
  QueueBase base=(QueueBase)objects[0];
  QueueLeaf recovered=(QueueLeaf)base;
  QueueTag tagged=recovered;
  if(recovered==leaf)result+=tagged.value()+base.value();
  Object absent=null;QueueLeaf missing=(QueueLeaf)absent;
  if(missing==null)result+=100;
  Object other=new QueueOther();
  try{QueueLeaf wrong=(QueueLeaf)other;result+=1000;}catch(ClassCastException expected){result+=10;}
  Object[] leaves=new QueueLeaf[]{leaf};
  try{leaves[0]=other;result+=10000;}catch(ArrayStoreException expected){result+=20;}
  Object nested=new int[][]{{1,2}};
  if(nested instanceof Object[])result+=40;
  if(nested instanceof Cloneable)result+=80;
  return result;
 }
}
