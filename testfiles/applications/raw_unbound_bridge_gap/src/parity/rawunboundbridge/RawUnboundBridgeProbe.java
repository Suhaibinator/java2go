package parity.rawunboundbridge;

public final class RawUnboundBridgeProbe {
    interface Numbered {
        int number();
    }

    static final class First implements Numbered {
        final int value;

        First(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }
    }

    static final class Second implements Numbered {
        final int value;

        Second(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }
    }

    static int baseBodies;
    static int specializedBodies;

    static class Base<T extends Numbered> {
        T value;

        Base(T value) {
            this.value = value;
        }

        T exchange(T next) {
            baseBodies++;
            T previous = value;
            value = next;
            return previous;
        }
    }

    static final class Specialized extends Base<First> {
        Specialized(First value) {
            super(value);
        }

        @Override
        First exchange(First next) {
            specializedBodies++;
            return super.exchange(next);
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        RawUnbound operation = Base::exchange;

        Base<First> plain = new Base<First>(new First(1));
        Numbered plainResult = operation.apply(plain, new Second(2));

        Specialized specialized = new Specialized(new First(3));
        Numbered specializedResult = operation.apply(specialized, new First(4));

        String rejected;
        try {
            operation.apply(specialized, new Second(5));
            rejected = "missing";
        } catch (ClassCastException expected) {
            rejected = "ClassCastException";
        }

        return plainResult.number() + ":" + specializedResult.number() + ":" +
                baseBodies + ":" + specializedBodies + ":" + rejected;
    }

    public static void main(String[] args) {
        System.out.println(run());
    }
}
