public final class AnonymousVolatileFieldWitness {
 static int trace;
 static int record(int digit){trace=trace*10+digit;return digit;}
 static class OrdinaryBase {int flag=40;OrdinaryBase(){record(1);}}
 static class VolatileBase {volatile long flag=40L;VolatileBase(){record(1);}}
 static int read(){trace=0;int value=(new OrdinaryBase(){volatile int flag=record(2);{record(3);}}).flag;return trace*100+value;}
 static int write(){trace=0;int value=((new OrdinaryBase(){volatile int flag=record(2);{record(3);}}).flag=record(4));return trace*100+value;}
 static int compound(){trace=0;int value=((new OrdinaryBase(){volatile int flag=record(2);{record(3);}}).flag+=record(4));return trace*100+value;}
 static int postfix(){trace=0;int value=(new OrdinaryBase(){volatile int flag=record(2);{record(3);}}).flag++;return trace*100+value;}
 static int ordinaryShadow(){trace=0;int value=(new VolatileBase(){int flag=record(2);{record(3);}}).flag;return trace*100+value;}
 static int differingTypes(){trace=0;int value=(new VolatileBase(){volatile int flag=record(2);{record(3);}}).flag;return trace*100+value;}
 static int ordinaryShadowWrite(){trace=0;int value=((new VolatileBase(){int flag=record(2);{record(3);}}).flag=record(4));return trace*100+value;}
 static int ordinaryShadowCompound(){trace=0;int value=((new VolatileBase(){int flag=record(2);{record(3);}}).flag+=record(4));return trace*100+value;}
 static int ordinaryShadowPostfix(){trace=0;int value=(new VolatileBase(){int flag=record(2);{record(3);}}).flag++;return trace*100+value;}
 static int differingTypesWrite(){trace=0;int value=((new VolatileBase(){volatile int flag=record(2);{record(3);}}).flag=record(4));return trace*100+value;}
 static int differingTypesCompound(){trace=0;int value=((new VolatileBase(){volatile int flag=record(2);{record(3);}}).flag+=record(4));return trace*100+value;}
 static int differingTypesPostfix(){trace=0;int value=(new VolatileBase(){volatile int flag=record(2);{record(3);}}).flag++;return trace*100+value;}
 public static long run(){return read()==12302 && write()==123404 && compound()==123406 && postfix()==12302 && ordinaryShadow()==12302 && differingTypes()==12302 && ordinaryShadowWrite()==123404 && ordinaryShadowCompound()==123406 && ordinaryShadowPostfix()==12302 && differingTypesWrite()==123404 && differingTypesCompound()==123406 && differingTypesPostfix()==12302 ? 9L : 0L;}
}
