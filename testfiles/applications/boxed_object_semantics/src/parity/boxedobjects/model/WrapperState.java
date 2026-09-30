package parity.boxedobjects.model;

public class WrapperState {
    public Boolean flag;
    public Byte small;
    public Short narrow;
    public Character character;
    public Integer integer;
    public Long wide;
    public Float single;
    public Double decimal;

    public boolean allNull() {
        return flag == null && small == null && narrow == null && character == null
                && integer == null && wide == null && single == null && decimal == null;
    }

    public void set(Boolean flag, Byte small, Short narrow, Character character,
                    Integer integer, Long wide, Float single, Double decimal) {
        this.flag = identity(flag);
        this.small = identity(small);
        this.narrow = identity(narrow);
        this.character = identity(character);
        this.integer = identity(integer);
        this.wide = identity(wide);
        this.single = identity(single);
        this.decimal = identity(decimal);
    }

    public String values() {
        return flag + ":" + small + ":" + narrow + ":" + character + ":"
                + integer + ":" + wide + ":" + single + ":" + decimal;
    }

    public static <T> T identity(T value) {
        return value;
    }

    public static <T extends Number> int numberValue(T value) {
        return value.intValue();
    }

    public static <T extends Number> T choose(boolean first, T left, T right) {
        return first ? left : right;
    }
}
