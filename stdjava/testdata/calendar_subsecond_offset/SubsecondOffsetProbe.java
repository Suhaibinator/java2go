import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.Date;
import java.util.GregorianCalendar;
import java.util.Locale;
import java.util.TimeZone;

public class SubsecondOffsetProbe {
    public static void main(String[] args) throws Exception {
        if (args.length != 0) throw new IllegalArgumentException("no arguments");
        Locale general = Locale.getDefault();
        Locale format = Locale.getDefault(Locale.Category.FORMAT);
        Locale display = Locale.getDefault(Locale.Category.DISPLAY);
        TimeZone previous = TimeZone.getDefault();
        try {
            Locale.setDefault(Locale.US);
            TimeZone.setDefault(TimeZone.getTimeZone("UTC"));
            int[] offsets = {-1001, -999, -1, 1, 999, 1001};
            long[] instants = {-1001L, -1L, 0L, 1L, 999L, 1000L};
            int row = 0;
            for (int offset : offsets) {
                for (long instant : instants) {
                    TimeZone zone = TimeZone.getTimeZone("UTC");
                    zone.setRawOffset(offset);
                    GregorianCalendar calendar = new GregorianCalendar(zone, Locale.US);
                    calendar.setTimeInMillis(instant);
                    SimpleDateFormat formatter = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.US);
                    formatter.setTimeZone(zone);
                    formatter.setLenient(false);
                    String text = formatter.format(new Date(instant));
                    ParsePosition position = new ParsePosition(0);
                    position.setErrorIndex(7);
                    Date parsed = formatter.parse(text, position);
                    System.out.println("case=" + row++ + "|offset=" + offset + "|instant=" + instant
                        + "|text=" + text
                        + "|fields=" + calendar.get(Calendar.YEAR) + "," + calendar.get(Calendar.MONTH)
                        + "," + calendar.get(Calendar.DAY_OF_MONTH) + "," + calendar.get(Calendar.HOUR_OF_DAY)
                        + "," + calendar.get(Calendar.MINUTE) + "," + calendar.get(Calendar.SECOND)
                        + "," + calendar.get(Calendar.MILLISECOND) + "," + calendar.get(Calendar.ZONE_OFFSET)
                        + "," + calendar.get(Calendar.DST_OFFSET)
                        + "|parsed=" + (parsed == null ? "null" : Long.toString(parsed.getTime()))
                        + "|index=" + position.getIndex() + "|error=" + position.getErrorIndex()
                        + "|raw=" + zone.getRawOffset() + "|total=" + zone.getOffset(instant));
                }
            }
        } finally {
            Locale.setDefault(general);
            Locale.setDefault(Locale.Category.FORMAT, format);
            Locale.setDefault(Locale.Category.DISPLAY, display);
            TimeZone.setDefault(previous);
        }
    }
}
