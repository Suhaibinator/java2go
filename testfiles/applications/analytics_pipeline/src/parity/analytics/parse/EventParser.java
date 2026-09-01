package parity.analytics.parse;

import parity.analytics.common.ParseResult;
import parity.analytics.model.Event;

public class EventParser implements RecordParser<Event> {
    public ParseResult<Event> parse(String line) {
        String[] fields = line.split("\\|");
        if (fields.length != 6) {
            return new ParseResult<Event>(false, null, "FIELD_COUNT", "expected_6_fields_got_" + fields.length);
        }

        String id = fields[0].trim();
        String segment = fields[1].trim().toLowerCase();
        String action = fields[2].trim().toUpperCase();
        String latencyText = fields[3].trim();
        String successText = fields[4].trim().toLowerCase();
        String unitsText = fields[5].trim();

        if (!id.startsWith("EVT-") || id.length() < 8) {
            return new ParseResult<Event>(false, null, "INVALID_ID", "id_" + id);
        }
        if (segment.length() == 0) {
            return new ParseResult<Event>(false, null, "EMPTY_SEGMENT", "segment_blank");
        }
        if (!isKnownAction(action)) {
            return new ParseResult<Event>(false, null, "INVALID_ACTION", "action_" + action);
        }
        if (!isUnsignedInteger(latencyText)) {
            return new ParseResult<Event>(false, null, "INVALID_NUMBER", "latency_" + latencyText);
        }
        if (!successText.equals("true") && !successText.equals("false")) {
            return new ParseResult<Event>(false, null, "INVALID_BOOLEAN", "success_" + successText);
        }
        if (!isSignedInteger(unitsText)) {
            return new ParseResult<Event>(false, null, "INVALID_NUMBER", "units_" + unitsText);
        }

        int latency = Integer.parseInt(latencyText);
        int units = Integer.parseInt(unitsText);
        if (units < 0) {
            return new ParseResult<Event>(false, null, "NEGATIVE_UNITS", "units_" + units);
        }

        boolean successful = Boolean.parseBoolean(successText);
        Event event = new Event(id, segment, action, latency, successful, units);
        return new ParseResult<Event>(true, event, "", "");
    }

    private boolean isKnownAction(String action) {
        switch (action) {
            case "VIEW":
            case "SEARCH":
            case "CART":
            case "PURCHASE":
            case "REFUND":
            case "SUPPORT":
                return true;
            default:
                return false;
        }
    }

    private boolean isUnsignedInteger(String value) {
        if (value.length() == 0) {
            return false;
        }
        for (int i = 0; i < value.length(); i++) {
            char current = value.charAt(i);
            if (!Character.isDigit(current)) {
                return false;
            }
        }
        return true;
    }

    private boolean isSignedInteger(String value) {
        if (value.length() == 0) {
            return false;
        }
        int start = 0;
        if (value.charAt(0) == '-') {
            if (value.length() == 1) {
                return false;
            }
            start = 1;
        }
        for (int i = start; i < value.length(); i++) {
            if (!Character.isDigit(value.charAt(i))) {
                return false;
            }
        }
        return true;
    }
}
