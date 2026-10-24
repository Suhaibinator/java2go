package transpiler

import "testing"

func TestCampaignMapEntryGenericSourceStorageJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>entry</groupId><artifactId>generic</artifactId><version>1</version></project>`,
		"src/main/java/probe/genericentry/Main.java": `package probe.genericentry;

import java.util.Map;

public final class Main {
    // Bare generic fields and a self-family reference mirror the representation
    // pressure in real source Nodes; this is an application entry, not a JDK replacement.
    static final class Node<K, V> implements Map.Entry<K, V> {
        final K key;
        V value;
        Node<K, V> next;
        int reads;
        int writes;
        Node(K key, V value) { this.key = key; this.value = value; }
        public K getKey() { return key; }
        public V getValue() { reads++; return value; }
        public V setValue(V nextValue) {
            V old = value;
            value = nextValue;
            writes++;
            return old;
        }
    }
    static <K, V> Map.Entry<K, V> pass(Node<K, V> source) { return source; }
    static void require(boolean ok, String label) { if (!ok) throw new AssertionError(label); }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        Node<String, String> source = new Node<String, String>("key", "before");
        Map.Entry<String, String> typed = pass(source);
        Map.Entry raw = typed;
        Object alias = typed;
        Node rawNode = source;
        Node<Object, Object> objectNode = (Node<Object, Object>) rawNode;
        boolean identity = alias == source && typed == raw && rawNode == objectNode && alias instanceof Map.Entry;
        require(identity, "one source entry through erased and typed aliases");
        System.out.println("identity=" + identity + ":" + typed.getKey());

        String old = typed.setValue("first");
        raw.setValue(Integer.valueOf(17));
        boolean oldCastFailed = false;
        try { String wrongOld = typed.setValue("replacement"); wrongOld.length(); }
        catch (ClassCastException expected) { oldCastFailed = true; }
        Object afterWrite = raw.getValue();
        String recovered = typed.getValue();
        require(old.equals("before") && oldCastFailed && source.writes == 3
                && afterWrite.equals("replacement") && recovered.equals("replacement") && source.reads == 2,
                "old-value checkcast must follow mutation and counter");
        System.out.println("oldcast=" + old + ":" + oldCastFailed + ":" + afterWrite + ":"
                + recovered + ":" + source.writes + ":" + source.reads);

        raw.setValue(Integer.valueOf(19));
        Object wrongValue = objectNode.value;
        boolean readCastFailed = false;
        try { String bad = typed.getValue(); bad.length(); }
        catch (ClassCastException expected) { readCastFailed = true; }
        raw.setValue(null);
        String nullable = typed.getValue();
        require(wrongValue.equals(Integer.valueOf(19)) && readCastFailed && nullable == null
                && source.writes == 5 && source.reads == 4, "Object field read and delayed typed get failure");
        System.out.println("readcast=" + wrongValue + ":" + readCastFailed + ":" + (nullable == null)
                + ":" + source.writes + ":" + source.reads);

        Node<String, String> retained = new Node<String, String>("tail", "next");
        source.next = retained;
        boolean originalLink = (Object) objectNode.next == retained;
        objectNode.next = objectNode;
        boolean selfLink = source.next == source && retained != source && retained.next == null;
        require(originalLink && selfLink, "self-family references share erased storage");
        System.out.println("links=" + originalLink + ":" + selfLink + ":" + retained.getKey());

        Map.Entry<String, String> missing = null;
        boolean nullFailure = false;
        try { missing.setValue("ignored"); } catch (NullPointerException expected) { nullFailure = true; }
        System.out.println("null=" + nullFailure);
    }
}
`,
	}, "probe.genericentry.Main", "identity=true:key\noldcast=before:true:replacement:replacement:3:2\nreadcast=19:true:true:5:4\nlinks=true:true:tail\nnull=true\n")
}
