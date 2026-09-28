package probe;

import com.google.gson.internal.bind.util.ISO8601Utils;
import java.text.ParseException;
import java.text.ParsePosition;
import java.util.Date;
import java.util.TimeZone;

public final class Main {
  static void parse(String input, int start) {
    ParsePosition position = new ParsePosition(start);
    try {
      Date value = ISO8601Utils.parse(input, position);
      System.out.println(input + "|" + value.getTime() + "|" + position.getIndex() + "|" + position.getErrorIndex());
    } catch (ParseException failure) {
      System.out.println(input + "|error|" + position.getIndex() + "|" + position.getErrorIndex()
          + "|" + failure.getErrorOffset() + "|" + failure.getCause().getClass().getSimpleName()
          + "|" + failure.getMessage());
    }
  }
  public static void main(String[] args) {
    TimeZone.setDefault(TimeZone.getTimeZone("UTC"));
    String[] inputs = {
      "2024-02-29", "1970-01-01T00:00:00Z", "1969-12-31T23:59:59.999Z",
      "2024-02-29T12:34:56.1Z", "2024-02-29T12:34:56.12Z",
      "2024-02-29T12:34:56.123456Z", "20240229T1234Z",
      "2024-02-29T12:34:60Z", "2024-02-29T12:34:56+05:45",
      "2024-02-29T12:34:56-0330", "2024-02-29T12:34:56+05",
      "2023-02-29", "2023-02-29T00:00:00Z", "2024-13-01T00:00:00Z",
      "2024-02-29T12:34:56+25:00", "2024-02-29T12:34:56", "bad", "1582-10-10"
    };
    for (String input : inputs) { parse(input, 0); }
    parse("prefix2024-02-29T12:34:56Ztail", 6);
    System.out.println(ISO8601Utils.format(new Date(-1L)));
    System.out.println(ISO8601Utils.format(new Date(-1L), true));
    System.out.println(ISO8601Utils.format(new Date(1719792000123L), true, TimeZone.getTimeZone("America/New_York")));
    System.out.println(ISO8601Utils.format(new Date(0L), true, TimeZone.getTimeZone("GMT+0545")));
  }
}
