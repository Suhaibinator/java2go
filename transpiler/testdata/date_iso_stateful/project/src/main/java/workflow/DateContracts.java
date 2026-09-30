package workflow;

import com.google.gson.internal.bind.util.ISO8601Utils;
import java.text.DateFormat;
import java.text.ParseException;
import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Locale;
import java.util.TimeZone;

public final class DateContracts {
    public static void run(int seed) throws ParseException {
        Locale original = Locale.getDefault();
        Locale originalFormat = Locale.getDefault(Locale.Category.FORMAT);
        Locale originalDisplay = Locale.getDefault(Locale.Category.DISPLAY);
        TimeZone originalZone = TimeZone.getDefault();
        try {
            Locale.setDefault(Locale.US);
            Locale.setDefault(Locale.Category.FORMAT, Locale.US);
            TimeZone.setDefault(TimeZone.getTimeZone("UTC"));
            long base = 1615703400000L + (seed % 3) * 60000L;
            Date value = new Date(base);
            System.out.println("clock:" + seed + ":" + base);
            int[] styles = new int[] {DateFormat.SHORT, DateFormat.MEDIUM, DateFormat.LONG, DateFormat.FULL};
            for (int style : styles) {
                DateFormat explicit = DateFormat.getDateTimeInstance(style, style, Locale.US);
                DateFormat defaults = DateFormat.getDateTimeInstance(style, style);
                explicit.setLenient(false);
                java.lang.String text = explicit.format(value);
                ParsePosition position = new ParsePosition(0);
                Date parsed = explicit.parse(text, position);
                boolean sameDefault = defaults.format(value).equals(text);
                boolean sameInstant = parsed != null && parsed.getTime() == base;
                if (!sameDefault || !sameInstant || position.getIndex() != text.length()) throw new AssertionError("style round trip");
                System.out.println("style:" + style + ":" + text + ":" + sameDefault + ":" + sameInstant + ":" + position.getIndex() + ":" + position.getErrorIndex());
            }
            DateFormat mixed = DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.LONG, Locale.US);
            System.out.println("mixed:" + mixed.format(value) + ":" + mixed.getTimeZone().getID());

            DateFormat snapshot = DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.MEDIUM);
            java.lang.String before = snapshot.format(value);
            Locale.setDefault(Locale.Category.FORMAT, Locale.ENGLISH);
            Locale.setDefault(Locale.Category.DISPLAY, Locale.ENGLISH);
            System.out.println("locale:" + Locale.getDefault().getLanguage() + ":" + Locale.getDefault().getCountry() + ":" + Locale.getDefault(Locale.Category.FORMAT).getLanguage() + ":" + Locale.getDefault(Locale.Category.FORMAT).getCountry() + ":" + Locale.getDefault(Locale.Category.DISPLAY).getLanguage() + ":" + Locale.getDefault(Locale.Category.DISPLAY).getCountry());
            TimeZone.setDefault(TimeZone.getTimeZone("America/New_York"));
            DateFormat fresh = DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.MEDIUM);
            boolean held = snapshot.format(value).equals(before) && snapshot.getTimeZone().getID().equals("UTC");
            if (!held || !fresh.getTimeZone().getID().equals("America/New_York")) throw new AssertionError("formatter default snapshot");
            System.out.println("snapshot:" + held + ":" + snapshot.format(value) + ":" + fresh.format(value) + ":" + fresh.getTimeZone().getID());

            TimeZone passed = TimeZone.getTimeZone("GMT+02:00");
            snapshot.setTimeZone(passed);
            java.lang.String zoneBefore = snapshot.format(value);
            boolean zoneIdentity = snapshot.getTimeZone() == passed;
            passed.setRawOffset(3 * 3600000);
            java.lang.String zoneAfter = snapshot.format(value);
            snapshot.setTimeZone(TimeZone.getTimeZone("UTC"));
            System.out.println("timezone:" + zoneBefore + ":" + zoneAfter + ":" + zoneIdentity + ":" + snapshot.format(value));

            SimpleDateFormat parser = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US);
            parser.setTimeZone(TimeZone.getTimeZone("UTC"));
            parser.setLenient(false);
            java.lang.String canonical = parser.format(value);
            ParsePosition reusable = new ParsePosition(2);
            reusable.setErrorIndex(27);
            Date embedded = parser.parse("xx" + canonical + "tail", reusable);
            int successfulIndex = reusable.getIndex();
            int successfulError = reusable.getErrorIndex();
            reusable.setIndex(0);
            reusable.setErrorIndex(-1);
            Date invalid = parser.parse("not-a-date", reusable);
            int invalidIndex = reusable.getIndex();
            int invalidError = reusable.getErrorIndex();
            reusable.setIndex(0);
            Date recoveredWithStaleError = parser.parse(canonical, reusable);
            int staleError = reusable.getErrorIndex();
            reusable.setIndex(0);
            reusable.setErrorIndex(-1);
            Date recovered = parser.parse(canonical, reusable);
            boolean partialParse = parser.parse(canonical + "tail").getTime() == base;
            if (embedded == null || embedded.getTime() != base || invalid != null || invalidIndex != 0 || invalidError < 0 || recoveredWithStaleError == null || recovered == null || recovered.getTime() != base || !partialParse) throw new AssertionError("parse position state");
            System.out.println("position:" + successfulIndex + ":" + successfulError + ":" + invalidIndex + ":" + invalidError + ":" + staleError + ":" + reusable.getIndex() + ":" + reusable.getErrorIndex() + ":" + partialParse);

            java.lang.String failure = "missing";
            try { parser.parse("2021-02-29 12:00:00"); }
            catch (ParseException expected) { failure = expected.getClass().getName() + ":" + expected.getErrorOffset() + ":" + causeName(expected); }
            boolean invalidStyle = false;
            try { DateFormat.getDateTimeInstance(9, DateFormat.SHORT, Locale.US); }
            catch (IllegalArgumentException expected) { invalidStyle = true; }
            if (failure.equals("missing") || !invalidStyle) throw new AssertionError("invalid DateFormat contract");
            System.out.println("invalid:" + failure + ":" + invalidStyle);
            daylightWorkflow(base);
            isoWorkflow(value);
        } finally {
            Locale.setDefault(original);
            Locale.setDefault(Locale.Category.FORMAT, originalFormat);
            Locale.setDefault(Locale.Category.DISPLAY, originalDisplay);
            TimeZone.setDefault(originalZone);
        }
        boolean generalRestored = Locale.getDefault().equals(original);
        boolean formatRestored = Locale.getDefault(Locale.Category.FORMAT).equals(originalFormat);
        boolean displayRestored = Locale.getDefault(Locale.Category.DISPLAY).equals(originalDisplay);
        boolean zoneRestored = TimeZone.getDefault().getID().equals(originalZone.getID()) && TimeZone.getDefault().hasSameRules(originalZone);
        if (!generalRestored || !formatRestored || !displayRestored || !zoneRestored) throw new AssertionError("defaults not restored");
        System.out.println("restore:" + generalRestored + ":" + formatRestored + ":" + displayRestored + ":" + zoneRestored);
    }

    private static void daylightWorkflow(long base) {
        TimeZone ny = TimeZone.getTimeZone("America/New_York");
        SimpleDateFormat local = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US);
        local.setTimeZone(ny);
        local.setLenient(false);
        Date before = new Date(base);
        Date after = new Date(base + 3600000L);
        ParsePosition gapPosition = new ParsePosition(0);
        Date gap = local.parse("2021-03-14 02:30:00", gapPosition);
        ParsePosition overlapPosition = new ParsePosition(0);
        Date overlap = local.parse("2021-11-07 01:30:00", overlapPosition);
        if (gap != null || overlap == null || ny.getOffset(before.getTime()) == ny.getOffset(after.getTime())) throw new AssertionError("DST transition contract");
        System.out.println("dst:" + local.format(before) + ":" + local.format(after) + ":" + ny.getOffset(before.getTime()) + ":" + ny.getOffset(after.getTime()) + ":" + gapPosition.getIndex() + ":" + gapPosition.getErrorIndex() + ":" + overlap.getTime() + ":" + overlapPosition.getIndex());
    }

    private static void isoWorkflow(Date value) throws ParseException {
        Date springEarly = ISO8601Utils.parse("2021-03-14T01:30:00-05:00", new ParsePosition(0));
        Date springLate = ISO8601Utils.parse("2021-03-14T03:30:00-04:00", new ParsePosition(0));
        Date fallEarly = ISO8601Utils.parse("2021-11-07T01:30:00-04:00", new ParsePosition(0));
        Date fallLate = ISO8601Utils.parse("2021-11-07T01:30:00-05:00", new ParsePosition(0));
        Date explicitGap = ISO8601Utils.parse("2021-03-14T02:30:00-05:00", new ParsePosition(0));
        if (springLate.getTime() - springEarly.getTime() != 3600000L || fallLate.getTime() - fallEarly.getTime() != 3600000L) throw new AssertionError("explicit offset instants");
        System.out.println("iso-offsets:" + springEarly.getTime() + ":" + springLate.getTime() + ":" + fallEarly.getTime() + ":" + fallLate.getTime() + ":" + explicitGap.getTime());
        Date leap = ISO8601Utils.parse("2016-12-31T23:59:60Z", new ParsePosition(0));
        SimpleDateFormat strict = new SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ssX", Locale.US);
        strict.setLenient(false);
        ParsePosition strictPosition = new ParsePosition(0);
        Date strictLeap = strict.parse("2016-12-31T23:59:60Z", strictPosition);
        System.out.println("iso-leap:" + ISO8601Utils.format(leap, true) + ":" + (strictLeap == null) + ":" + strictPosition.getIndex() + ":" + strictPosition.getErrorIndex());
        ParsePosition prefix = new ParsePosition(2);
        Date prefixed = ISO8601Utils.parse("xx2021-03-14T06:30:00.12Ztail", prefix);
        System.out.println("iso-position:" + prefixed.getTime() + ":" + prefix.getIndex() + ":" + prefix.getErrorIndex() + ":" + ISO8601Utils.format(value, true, TimeZone.getTimeZone("America/New_York")));
        java.lang.String[] invalids = new java.lang.String[] {"2021-02-29T12:00:00Z", "2021-03-14T12:00:00+25:00", "2021-03-14T12:00:00"};
        for (java.lang.String invalid : invalids) {
            ParsePosition position = new ParsePosition(0);
            boolean rejected = false;
            try { ISO8601Utils.parse(invalid, position); }
            catch (ParseException expected) {
                rejected = true;
                System.out.println("iso-invalid:" + invalid + ":" + expected.getErrorOffset() + ":" + position.getIndex() + ":" + position.getErrorIndex() + ":" + causeName(expected));
            }
            if (!rejected) throw new AssertionError("invalid ISO accepted");
        }
    }

    private static java.lang.String causeName(Throwable error) {
        Throwable cause = error.getCause();
        return cause == null ? "none" : cause.getClass().getName();
    }
}
