import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;

public class Comparators {
    // A Comparable user type, to prove Collections.sort reaches a generated
    // compareTo rather than only the built-in primitive orderings.
    static class Version implements Comparable<Version> {
        int major;
        int minor;

        Version(int major, int minor) {
            this.major = major;
            this.minor = minor;
        }

        public int compareTo(Version other) {
            if (this.major != other.major) {
                return this.major < other.major ? -1 : 1;
            }
            if (this.minor == other.minor) {
                return 0;
            }
            return this.minor < other.minor ? -1 : 1;
        }

        // Rendered explicitly rather than through toString(), which the generated
        // string bridge does not yet consult (KNOWN_ISSUES K17).
        String render() {
            return major + "." + minor;
        }
    }

    static String renderVersions(List<Version> versions) {
        StringBuilder out = new StringBuilder();
        out.append("[");
        for (int i = 0; i < versions.size(); i++) {
            if (i > 0) {
                out.append(", ");
            }
            out.append(versions.get(i).render());
        }
        out.append("]");
        return out.toString();
    }

    static String renderVersionArray(Version[] versions) {
        StringBuilder out = new StringBuilder();
        out.append("[");
        for (int i = 0; i < versions.length; i++) {
            if (i > 0) {
                out.append(", ");
            }
            out.append(versions[i].render());
        }
        out.append("]");
        return out.toString();
    }

    static String renderPeople(List<Person> people) {
        StringBuilder out = new StringBuilder();
        out.append("[");
        for (int i = 0; i < people.size(); i++) {
            if (i > 0) {
                out.append(", ");
            }
            out.append(people.get(i).render());
        }
        out.append("]");
        return out.toString();
    }

    static class Person {
        String name;
        int age;

        Person(String name, int age) {
            this.name = name;
            this.age = age;
        }

        String render() {
            return name + ":" + age;
        }
    }

    public static void main(String[] args) {
        List<Integer> nums = new ArrayList<Integer>();
        nums.add(5);
        nums.add(1);
        nums.add(4);
        nums.add(2);

        // Collections.sort with an inline comparator lambda.
        Collections.sort(nums, (a, b) -> a - b);
        System.out.println(nums);

        // Descending, to prove the comparator's sign is honoured.
        Collections.sort(nums, (a, b) -> b - a);
        System.out.println(nums);

        // Collections.max / min with a comparator.
        System.out.println("max " + Collections.max(nums, (a, b) -> a - b));
        System.out.println("min " + Collections.min(nums, (a, b) -> a - b));

        // List.sort with a comparator.
        nums.sort((a, b) -> a - b);
        System.out.println(nums);

        // Stability: equal keys must keep their input order. Sorting by age only,
        // the two 30-year-olds must stay in the order they were added.
        List<Person> people = new ArrayList<Person>();
        people.add(new Person("first", 30));
        people.add(new Person("second", 10));
        people.add(new Person("third", 30));
        people.add(new Person("fourth", 10));
        people.sort((a, b) -> a.age - b.age);
        System.out.println(renderPeople(people));

        // Collections.max / min keep the earlier element on ties.
        System.out.println("oldest " + Collections.max(people, (a, b) -> a.age - b.age).render());
        System.out.println("youngest " + Collections.min(people, (a, b) -> a.age - b.age).render());

        // Natural ordering of a user Comparable through Collections.sort.
        List<Version> versions = new ArrayList<Version>();
        versions.add(new Version(2, 0));
        versions.add(new Version(1, 9));
        versions.add(new Version(1, 2));
        Collections.sort(versions);
        System.out.println(renderVersions(versions));

        // ...and through Arrays.sort on a reference array.
        Version[] versionArray = new Version[3];
        versionArray[0] = new Version(3, 1);
        versionArray[1] = new Version(1, 0);
        versionArray[2] = new Version(2, 7);
        Arrays.sort(versionArray);
        System.out.println(renderVersionArray(versionArray));

        // Arrays.sort with an explicit comparator.
        String[] words = new String[4];
        words[0] = "pear";
        words[1] = "fig";
        words[2] = "banana";
        words[3] = "kiwi";
        Arrays.sort(words, (a, b) -> a.length() - b.length());
        System.out.println(Arrays.toString(words));

        // Comparator held in a variable, then reversed.
        Comparator<Integer> ascending = (a, b) -> a - b;
        Collections.sort(nums, ascending.reversed());
        System.out.println(nums);

        // thenComparing: age, then name, over a two-parameter comparator.
        List<Person> tied = new ArrayList<Person>();
        tied.add(new Person("carol", 30));
        tied.add(new Person("alice", 30));
        tied.add(new Person("bob", 20));
        Comparator<Person> byAge = (a, b) -> a.age - b.age;
        Comparator<Person> byName = (a, b) -> a.name.compareTo(b.name);
        tied.sort(byAge.thenComparing(byName));
        System.out.println(renderPeople(tied));

        // compare() called directly on a comparator value.
        System.out.println("compare " + byAge.compare(tied.get(0), tied.get(2)));
    }
}
