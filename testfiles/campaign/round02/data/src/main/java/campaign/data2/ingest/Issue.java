package campaign.data2.ingest;

public final class Issue {
    public final String location;
    public final String code;
    public final String fingerprint;

    public Issue(String location, String code, String fingerprint) {
        this.location = location;
        this.code = code;
        this.fingerprint = fingerprint;
    }

    public String row() { return location + "\t" + code + "\t" + fingerprint; }
}
