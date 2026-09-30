package parity.dependentgeneric;

public final class DependentGenericConversionApplication {
    interface Root {
        int value();
    }

    static class Base implements Root {
        private final int value;

        Base(int value) {
            this.value = value;
        }

        public int value() {
            return value;
        }

        int baseCode() {
            return value * 10 + 1;
        }
    }

    static final class Impl extends Base {
        private final int detail;

        Impl(int value, int detail) {
            super(value);
            this.detail = detail;
        }

        int detail() {
            return detail;
        }
    }

    static <B extends Base, T extends B> B widen(T value) {
        return value;
    }

    static <B extends Base> int combine(B first, B second) {
        return first.baseCode() * 100 + second.baseCode();
    }

    static <B extends Base, T extends B> int dependentView(B anchor, T concrete) {
        B widened = concrete;
        return combine(anchor, widened);
    }

    static <B extends Base, T extends B> int nestedInference(B anchor, T concrete) {
        return combine(anchor, widen(concrete));
    }

    static final class ConstructorProbe {
        private final int score;

        <B extends Base, T extends B> ConstructorProbe(B anchor, T concrete) {
            B widened = concrete;
            score = anchor.baseCode() * 1000
                    + widened.baseCode() * 10
                    + concrete.value();
        }

        int score() {
            return score;
        }
    }

    public static void main(String[] args) {
        Base explicitAnchor = new Base(2);
        Impl explicitValue = new Impl(3, 30);
        int explicitView = DependentGenericConversionApplication
                .<Base, Impl>dependentView(explicitAnchor, explicitValue);

        Base implicitAnchor = new Base(4);
        Impl implicitValue = new Impl(5, 50);
        int implicitView = dependentView(implicitAnchor, implicitValue);

        Base nestedAnchor = new Base(6);
        Impl nestedValue = new Impl(7, 70);
        int nested = nestedInference(nestedAnchor, nestedValue);

        Base explicitConstructorAnchor = new Base(8);
        Impl explicitConstructorValue = new Impl(9, 90);
        ConstructorProbe explicitConstructor = new <Base, Impl> ConstructorProbe(
                explicitConstructorAnchor, explicitConstructorValue);

        Base implicitConstructorAnchor = new Base(1);
        Impl implicitConstructorValue = new Impl(2, 20);
        ConstructorProbe implicitConstructor = new ConstructorProbe(
                implicitConstructorAnchor, implicitConstructorValue);

        int checksum = explicitView * 3 + implicitView * 5 + nested * 7
                + explicitConstructor.score() * 11
                + implicitConstructor.score() * 13
                + explicitValue.detail() + implicitValue.detail()
                + nestedValue.detail() + explicitConstructorValue.detail()
                + implicitConstructorValue.detail();

        System.out.println("VIEWS=" + explicitView + ":" + implicitView);
        System.out.println("NESTED=" + nested);
        System.out.println("CONSTRUCTORS=" + explicitConstructor.score()
                + ":" + implicitConstructor.score());
        System.out.println("CHECKSUM=" + checksum);
    }
}
