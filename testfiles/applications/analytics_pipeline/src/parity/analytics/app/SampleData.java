package parity.analytics.app;

public class SampleData {
    public static String[] lines() {
        return new String[] {
                "EVT-1001|north|VIEW|120|true|2",
                "EVT-1002| south |search|230|TRUE|1",
                "EVT-1003|north|CART|310|false|3",
                "EVT-1004|west|PURCHASE|450|true|5",
                "EVT-1005|south|VIEW|80|true|1",
                "broken|row",
                "EVT-1006|north|PURCHASE|190|true|4",
                "EVT-1007|west|REFUND|520|false|2",
                "EVT-1008|south|SUPPORT|275|true|2",
                "EVT-1009|west|PURCHASE|oops|true|4",
                "EVT-1010|east|SEARCH|160|false|1",
                "EVT-1011|east|CART|140|true|2",
                "EVT-1012|west|VIEW|90|true|3",
                "EVT-1013|north|SUPPORT|410|true|1",
                "EVT-1014|east|PURCHASE|360|true|6",
                "EVT-1015|south|VIEW|95|maybe|1",
                "EVT-1016|central|PURCHASE|210|true|-2",
                "EVT-1017|central|SEARCH|110|true|2",
                "EVT-1018|central|CART|330|false|2",
                "EVT-1019|north|UNKNOWN|100|true|1"
        };
    }
}
