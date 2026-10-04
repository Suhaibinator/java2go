class Parent {}
public class LocalTypeShadow {
    public static void main(String[] args) {
        Parent parent = new Parent();
        Object broad = parent;
        Parent copy = (Parent) broad;
        System.out.println(copy == parent);
    }
}
