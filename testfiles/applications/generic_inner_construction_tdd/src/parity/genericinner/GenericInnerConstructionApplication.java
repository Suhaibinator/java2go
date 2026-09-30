package parity.genericinner;

public final class GenericInnerConstructionApplication {
    interface Numbered {
        int number();
    }

    static final class Item implements Numbered {
        private final int number;

        Item(int number) {
            this.number = number;
        }

        public int number() {
            return number;
        }
    }

    static class GenericOuter<T extends Numbered> {
        T outerValue;

        GenericOuter(T outerValue) {
            this.outerValue = outerValue;
        }

        class Inner<U extends Numbered> {
            U innerValue;

            Inner(U innerValue) {
                this.innerValue = innerValue;
            }

            int mutateAndRead(T nextOuter, U nextInner) {
                int before = outerValue.number() * 10 + innerValue.number();
                GenericOuter.this.outerValue = nextOuter;
                this.innerValue = nextInner;
                int after = outerValue.number() * 10 + innerValue.number();
                return before * 100 + after;
            }
        }
    }

    static final class Derived<T extends Numbered> extends GenericOuter<T> {
        Derived(T outerValue) {
            super(outerValue);
        }

        int constructImplicitly(T nextOuter, Item initialInner, Item nextInner) {
            Inner<Item> inner = new Inner<Item>(initialInner);
            return inner.mutateAndRead(nextOuter, nextInner);
        }

        int constructExplicitly(T nextOuter, Item initialInner, Item nextInner) {
            GenericOuter<T>.Inner<Item> inner = this.new Inner<Item>(initialInner);
            return inner.mutateAndRead(nextOuter, nextInner);
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    private static int rawMutation(GenericOuter rawOuter) {
        GenericOuter.Inner rawInner = rawOuter.new Inner(new Item(2));
        return rawInner.mutateAndRead(new Item(3), new Item(4));
    }

    public static void main(String[] args) {
        GenericOuter rawOuter = new GenericOuter(new Item(1));
        int raw = rawMutation(rawOuter);

        Derived<Item> derived = new Derived<Item>(new Item(5));
        int implicit = derived.constructImplicitly(new Item(7), new Item(6), new Item(8));
        int explicit = derived.constructExplicitly(new Item(1), new Item(9), new Item(2));

        int rawFinal = ((Numbered) rawOuter.outerValue).number();
        int derivedFinal = derived.outerValue.number();
        int checksum = raw * 31 + implicit * 17 + explicit * 13
                + rawFinal * 7 + derivedFinal;

        System.out.println("RAW=" + raw);
        System.out.println("INHERITED=" + implicit + ":" + explicit);
        System.out.println("FINAL=" + rawFinal + ":" + derivedFinal);
        System.out.println("CHECKSUM=" + checksum);
    }
}
