package campaign.business2.dispatch;

public final class DispatchFailure extends Exception {
    public DispatchFailure(String code) {
        super(code);
    }
}
