import java.util.Optional;

public class Optionals {
    static Optional<String> find(int id) {
        if (id == 1) {
            return Optional.of("alice");
        }
        return Optional.empty();
    }

    public static void main(String[] args) {
        Optional<String> a = find(1);
        Optional<String> b = find(2);

        System.out.println(a.isPresent());
        System.out.println(b.isPresent());
        System.out.println(a.get());
        System.out.println(b.orElse("nobody"));
        System.out.println(a.orElse("nobody"));

        Optional<Integer> num = Optional.of(10);
        System.out.println(num.map(n -> n * 2).get());
    }
}
