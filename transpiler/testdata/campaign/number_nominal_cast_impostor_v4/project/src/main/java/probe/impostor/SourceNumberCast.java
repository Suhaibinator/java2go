package probe.impostor;

import probe.shadow.Number;

// The imported source Number must retain its source class identity.
public final class SourceNumberCast {
    public static Number cast(Object value) { return (Number) value; }
}
