public class OopBasic {
    private int value;
    private String label;

    public OopBasic(int value, String label) {
        this.value = value;
        this.label = label;
    }

    public int getValue() {
        return value;
    }

    public void setValue(int value) {
        this.value = value;
    }

    public String describe() {
        return label + "=" + value;
    }

    public int doubled() {
        return value * 2;
    }

    public static void main(String[] args) {
        OopBasic o = new OopBasic(10, "count");
        System.out.println(o.describe());
        System.out.println(o.getValue());
        System.out.println(o.doubled());
        o.setValue(25);
        System.out.println(o.describe());
        System.out.println(o.doubled());
    }
}
