package campaign.data.model;

public final class Reject {
    public final String location;
    public final String code;
    public final String fingerprint;

    public Reject(String location, String code, String fingerprint) {
        this.location = location;
        this.code = code;
        this.fingerprint = fingerprint;
    }

    public String toLine() {
        return location + "\t" + code + "\t" + fingerprint;
    }
}
