package transpiler

import "testing"

// These controls must validate their authored observations on JDK21 before any
// generated result is classified. They intentionally exercise public Java
// boundaries rather than handwritten calls to the inactive Go String core.
func TestCampaignStringABILiteralIdentityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-abi-literals</artifactId><version>1</version></project>`,
		"src/main/java/abiidentity/Entry.java": `package abiidentity;
public class Entry {
    static final String PREFIX = "A";
    static final String CONSTANT = PREFIX + "B";
    static String runtime(String value) { return value; }
    public static void main(String[] args) {
        String literal = "AB";
        final String local = "A";
        String copied = new String(literal);
        String value = runtime("AB");
        System.out.println("literal=" + (literal == "AB") + ":" + (literal == CONSTANT) + ":" + (literal == local + "B"));
        System.out.println("copy=" + (copied == literal) + ":" + copied.equals(literal) + ":" + (new String() == ""));
        System.out.println("runtime=" + (value + "" == value) + ":" + ("" + value == value) + ":" + (value.concat("") == value));
        System.out.println("slice=" + (copied.substring(0) == copied) + ":" + (copied.substring(1, 1) == ""));
    }
}`,
	}, "abiidentity.Entry", "literal=true:true:true\ncopy=false:true:false\nruntime=false:false:true\nslice=true:true\n")
}

func TestCampaignStringABIUTF16LiteralJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-abi-utf16-literal</artifactId><version>1</version></project>`,
		"src/main/java/abiliteral/Entry.java": `package abiliteral;
public class Entry {
    public static void main(String[] args) {
        String isolated = "x\uD83Dy\uDE00";
        String pair = "\uD83D\uDE00";
        String escaped = "\\uD800";
        char high = '\uD83D';
        System.out.println("isolated=" + isolated.length() + ":" + (int)isolated.charAt(0) + ":" + (int)isolated.charAt(1) + ":" + (int)isolated.charAt(2) + ":" + (int)isolated.charAt(3));
        System.out.println("pair=" + pair.length() + ":" + (int)pair.charAt(0) + ":" + (int)pair.charAt(1));
        System.out.println("escaped=" + escaped.length() + ":" + (int)escaped.charAt(0));
        System.out.println("char=" + (int)high + ":" + ("" + high).length() + ":" + (int)("" + high).charAt(0));
    }
}`,
	}, "abiliteral.Entry", "isolated=4:120:55357:121:56832\npair=2:55357:56832\nescaped=6:92\nchar=55357:1:55357\n")
}

func TestCampaignStringABINullStorageJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-abi-null</artifactId><version>1</version></project>`,
		"src/main/java/abinull/Entry.java": `package abinull;
public class Entry {
    String field;
    static String staticField;
    static String identity(String value) { return value; }
    static String absent() { return null; }
    public static void main(String[] args) {
        Entry entry = new Entry();
        String local = null;
        String selected = args.length == 0 ? local : "unused";
        String[] array = new String[2];
        String[][] matrix = new String[1][1];
        Object erased = selected;
        String restored = (String) erased;
        array[0] = identity(selected);
        array[1] = "";
        System.out.println("nulls=" + (entry.field == null) + ":" + (staticField == null) + ":" + (local == null) + ":" + (restored == null));
        System.out.println("arrays=" + (array[0] == null) + ":" + (array[1] != null) + ":" + (matrix[0][0] == null));
        System.out.println("results=" + (identity(null) == null) + ":" + (absent() == null) + ":" + String.valueOf(selected));
        local += "x";
        entry.field += "y";
        array[0] += "z";
        System.out.println("compound=" + local + ":" + entry.field + ":" + array[0]);
    }
}`,
	}, "abinull.Entry", "nulls=true:true:true:true\narrays=true:true:true\nresults=true:true:null\ncompound=nullx:nully:nullz\n")
}

func TestCampaignStringABIGenericReferenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-abi-generics</artifactId><version>1</version></project>`,
		"src/main/java/abigeneric/Entry.java": `package abigeneric;
class Box<T> {
    T value;
    T get() { return value; }
    void set(T value) { this.value = value; }
}
public class Entry {
    static <T> T identity(T value) { return value; }
    public static void main(String[] args) {
        Box<String> box = new Box<>();
        System.out.println("default=" + (box.get() == null));
        String original = new String("key");
        box.set(original);
        String[] strings = {original, null};
        Object[] erased = strings;
        String fromArray = (String) erased[0];
        String generic = identity(original);
        System.out.println("identity=" + (box.get() == original) + ":" + (fromArray == original) + ":" + (generic == original));
        System.out.println("distinct=" + (box.get() == new String("key")) + ":" + box.get().equals(new String("key")));
        System.out.println("null=" + (identity((String)null) == null) + ":" + (erased[1] == null));
        Box raw = box;
        raw.set(new Object());
        try {
            String polluted = box.get();
            System.out.println("wrong=" + polluted);
        } catch (ClassCastException expected) {
            System.out.println("polluted=cast");
        }
    }
}`,
	}, "abigeneric.Entry", "default=true\nidentity=true:true:true\ndistinct=false:true\nnull=true:true\npolluted=cast\n")
}

func TestCampaignStringABISwitchBoundaryJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-abi-switch</artifactId><version>1</version></project>`,
		"src/main/java/abiswitch/Entry.java": `package abiswitch;
public class Entry {
    static int trace;
    static String selector(String value) { trace++; return value; }
    static int statement(String value) {
        switch (selector(value)) {
            case "a" + "b": return 1;
            case "": return 2;
            default: return 3;
        }
    }
    static int expression(String value) {
        return switch (selector(value)) {
            case "ab" -> 4;
            case "" -> 5;
            default -> 6;
        };
    }
    public static void main(String[] args) {
        System.out.println("content=" + statement(new String("ab")) + ":" + expression(new String("ab")) + ":" + statement("") + ":" + expression("z"));
        try { statement(null); System.out.println("wrong-statement"); }
        catch (NullPointerException expected) { System.out.println("statement=null"); }
        try { expression(null); System.out.println("wrong-expression"); }
        catch (NullPointerException expected) { System.out.println("expression=null"); }
        System.out.println("trace=" + trace);
    }
}`,
	}, "abiswitch.Entry", "content=1:4:2:6\nstatement=null\nexpression=null\ntrace=6\n")
}

func TestCampaignStringABIInternJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-abi-intern</artifactId><version>1</version></project>`,
		"src/main/java/abiintern/Entry.java": `package abiintern;
public class Entry {
    public static void main(String[] args) {
        String first = new String(new char[]{'q','7','x'});
        String second = new String(new char[]{'q','7','x'});
        String canonical = first.intern();
        System.out.println("separate=" + (first == second) + ":" + first.equals(second));
        System.out.println("pool=" + (canonical == second.intern()) + ":" + (canonical == "q7x".intern()) + ":" + (canonical == canonical.intern()));
    }
}`,
	}, "abiintern.Entry", "separate=false:true\npool=true:true:true\n")
}
