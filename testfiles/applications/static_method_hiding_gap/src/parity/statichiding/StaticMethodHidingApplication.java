package parity.statichiding;

class StaticParent {
    static String kind() {
        return "parent";
    }
}

class StaticChild extends StaticParent {
    static String kind() {
        return "child";
    }
}

public final class StaticMethodHidingApplication {
    public static void main(String[] args) {
        System.out.println(StaticParent.kind() + ":" + StaticChild.kind());
    }
}
