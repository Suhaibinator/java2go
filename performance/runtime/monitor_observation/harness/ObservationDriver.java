public final class ObservationDriver {
    private static long heap() {
        System.gc(); System.gc();
        Runtime r=Runtime.getRuntime(); return r.totalMemory()-r.freeMemory();
    }
    public static void main(String[] args) throws Exception {
        System.out.println("ownership="+MonitorObservation.ownership());
        int mode=Integer.parseInt(args[0]);
        String.format("round=%d checksum=%d",0,0L);
        long baseline=heap();
        for(int round=0;round<3;round++) {
            long checksum=MonitorObservation.cycle(round,32,131072,mode);
            System.out.printf("round=%d checksum=%d%n",round,checksum);
            long current=heap();
            System.err.println("{\"round\":"+round+",\"delta\":"+(current-baseline)+"}");
        }
    }
}
