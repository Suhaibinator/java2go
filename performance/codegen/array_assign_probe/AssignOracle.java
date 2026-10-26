public class AssignOracle {
    static String trace;
    static int[] current;
    static int mode;
    static int[] receiver() { trace += "R"; if(mode==4) throw new IllegalStateException(); return current; }
    static int index(int value) { trace += "I"; if(mode==5) throw new IllegalStateException(); return value; }
    static int rhs() { trace += "V"; if(mode==3) throw new IllegalStateException(); if(mode==6) current=new int[]{99}; return 7; }
    static void run(String name, boolean nil, int idx, int m) {
        trace=""; mode=m; current=nil?null:new int[]{0}; int[] original=current;
        String outcome;
        try { outcome="value="+(receiver()[index(idx)]=rhs()); }
        catch(RuntimeException e) { outcome=e.getClass().getSimpleName(); }
        System.out.println(name+":"+trace+":"+outcome+":"+(original==null?"null":original[0])+":"+(current==null?"null":current[0]));
    }
    public static void main(String[] args) {
        run("valid",false,0,0); run("null",true,0,0); run("null-negative",true,-1,0);
        run("negative",false,-1,0); run("end",false,1,0);
        run("min",false,Integer.MIN_VALUE,0); run("max",false,Integer.MAX_VALUE,0);
        run("rhs-null",true,0,3); run("rhs-bounds",false,1,3);
        run("receiver-throws",false,0,4); run("index-throws",true,0,5); run("saved-target",false,0,6);
    }
}
