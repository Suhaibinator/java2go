package abs.integration.complex;

public abstract class BaseThing {
    protected int value;
    public BaseThing(int value) { this.value = value; }
    public int value() { return value; }
    public abstract String id();
    public abstract double compute(double a, double b);
    public String describe() { return id() + ":" + value; }
}

public abstract class MidThing extends BaseThing {
    protected String name;
    public MidThing(int value, String name) { super(value); this.name = name; }
    public abstract String id();
    public abstract double combine(double first, double second, double third);
    public String label() { return name + "-" + value(); }
}

public abstract class LeafThing extends MidThing {
    public LeafThing(int value, String name) { super(value, name); }
    public String describe() { return "leaf-" + super.describe(); }
}

public class ConcreteThing extends LeafThing {
    public ConcreteThing(int value, String name) { super(value, name); }
    public String id() { return "concrete-" + name; }
    public double compute(double a, double b) { return (a + b) * value(); }
    public double combine(double first, double second, double third) {
        double total = first + second + third;
        return total + compute(total, value());
    }
    public String label() { return "override-" + super.label(); }
}

public class AltConcreteThing extends MidThing {
    public AltConcreteThing(int value, String name) { super(value, name); }
    public String id() { return "alt-" + name; }
    public double compute(double a, double b) { return (a - b) * value(); }
    public double combine(double first, double second, double third) { return compute(first, second) + third; }
}
