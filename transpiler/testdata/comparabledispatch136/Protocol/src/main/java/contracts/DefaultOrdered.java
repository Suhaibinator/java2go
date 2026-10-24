package contracts; import model.Item; public interface DefaultOrdered extends java.lang.Comparable<Item> {default int compareTo(Item value){app.Main.trace+="D;";return value==null?-33:60-value.key;}}
