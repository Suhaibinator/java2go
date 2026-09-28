public final class IdentityDriver {
    private static long heap(){System.gc();System.gc();Runtime r=Runtime.getRuntime();return r.totalMemory()-r.freeMemory();}
    public static void main(String[] args) {
        int seed=Integer.parseInt(args[0]),mode=Integer.parseInt(args[1]);
        for(int pass=0;pass<3;pass++) scenario(seed,mode,pass==2);
    }
    private static void scenario(int seed,int mode,boolean measured) {
        ObjectIdentityProbe.cycle(seed,4,1024,0); ObjectIdentityProbe.clear();
        String.format("round=%d checksum=%d",0,0L);long baseline=heap();
        for(int round=0;round<3;round++) {
            long checksum=ObjectIdentityProbe.cycle(seed+round*31,32,131072,mode);
            System.out.printf("round=%d checksum=%d%n",round,checksum);
            long current=heap();System.out.println("retained="+ObjectIdentityProbe.retainedChecksum());if(measured)System.err.println("{\"phase\":\"round-"+round+"\",\"delta\":"+(current-baseline)+"}");
        }
        ObjectIdentityProbe.clear();System.out.println("cleared");
        long current=heap();System.out.println("retained="+ObjectIdentityProbe.retainedChecksum());if(measured)System.err.println("{\"phase\":\"cleared\",\"delta\":"+(current-baseline)+"}");
    }
}
