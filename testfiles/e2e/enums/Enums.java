enum Day {
    MON, TUE, WED, THU, FRI, SAT, SUN;

    boolean isWeekend() {
        return this == SAT || this == SUN;
    }
}

enum Planet {
    EARTH(9.8), MARS(3.7);

    private final double gravity;

    Planet(double gravity) {
        this.gravity = gravity;
    }

    double weight(double mass) {
        return mass * gravity;
    }
}

public class Enums {
    public static void main(String[] args) {
        for (Day d : Day.values()) {
            System.out.println(d + " weekend=" + d.isWeekend());
        }
        System.out.println(Day.WED.ordinal());
        System.out.println(Day.valueOf("FRI"));
        System.out.println(Planet.EARTH.weight(10.0));
        System.out.println(Planet.MARS.weight(10.0));
    }
}
