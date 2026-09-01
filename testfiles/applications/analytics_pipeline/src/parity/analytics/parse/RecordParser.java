package parity.analytics.parse;

import parity.analytics.common.ParseResult;

public interface RecordParser<T> {
    ParseResult<T> parse(String line);
}
