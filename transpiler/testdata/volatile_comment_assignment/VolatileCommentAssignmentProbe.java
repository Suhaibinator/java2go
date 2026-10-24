public class VolatileCommentAssignmentProbe {
    volatile int value = 3;
    static int calls;
    int rhs() { calls++; return 4; }
    int update() { return (((value)) /* operator trivia */ += rhs()); }
    public static void main(String[] args) {
        VolatileCommentAssignmentProbe p = new VolatileCommentAssignmentProbe();
        int result = p.update();
        System.out.print(result + ":" + p.value + ":" + calls);
    }
}
