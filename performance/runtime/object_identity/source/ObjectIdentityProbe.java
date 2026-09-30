interface GraphTag { int value(); }
class GraphRoot {
    byte[] payload;
    GraphRoot peer;
    int number;
    GraphRoot(int number, int bytes) { this.number=number; payload=new byte[bytes]; }
    public int value() { return number; }
}
class GraphMiddle extends GraphRoot {
    GraphMiddle(int number, int bytes) { super(number, bytes); }
}
class GraphLeaf extends GraphMiddle implements GraphTag {
    GraphLeaf(int number, int bytes) { super(number, bytes); }
    @Override public int value() { return number+1; }
}

public final class ObjectIdentityProbe {
    private static GraphRoot retainedOne;
    private static GraphRoot[] retainedBatch;

    public static GraphLeaf make(int seed, int bytes) {
        GraphLeaf leaf=new GraphLeaf(seed,bytes);
        leaf.payload[0]=(byte)(seed & 127);
        leaf.peer=leaf;
        return leaf;
    }
    public static void clear() { retainedOne=null; retainedBatch=null; }
    public static long views(GraphLeaf leaf, int repeats) {
        GraphRoot[] array=new GraphRoot[]{leaf};
        long checksum=0;
        for(int i=0;i<repeats;i++) {
            GraphRoot base=array[0];
            GraphLeaf recovered=(GraphLeaf)base;
            GraphTag tagged=recovered;
            Object baseObject=base;
            Object leafObject=leaf;
            if(recovered!=leaf || tagged.value()!=base.value() || baseObject.getClass()!=leafObject.getClass()) {
                throw new AssertionError("identity/dispatch mismatch");
            }
            checksum+=tagged.value()+recovered.payload[0];
        }
        return checksum;
    }
    public static long retainedChecksum() {
        if(retainedOne!=null) { return views((GraphLeaf)retainedOne,1); }
        long checksum=0;
        if(retainedBatch!=null) {
            for(GraphRoot base:retainedBatch) { checksum+=views((GraphLeaf)base,1); }
        }
        return checksum;
    }
    public static long cycle(int seed,int count,int bytes,int mode) {
        GraphRoot[] batch=new GraphRoot[count];
        long checksum=0;
        for(int i=0;i<count;i++) {
            GraphLeaf leaf=make(seed+i,bytes);
            batch[i]=leaf;
            GraphRoot base=batch[i];
            GraphLeaf recovered=(GraphLeaf)base;
            if(recovered!=leaf) { throw new AssertionError("array identity mismatch"); }
            checksum+=views(recovered,3);
            if(mode==1) { retainedOne=base; }
        }
        if(mode==2) { retainedBatch=batch; }
        return checksum;
    }
}
