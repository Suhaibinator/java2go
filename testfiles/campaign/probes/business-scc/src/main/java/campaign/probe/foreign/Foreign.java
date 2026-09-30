package campaign.probe.foreign;

import campaign.probe.base.Base;

public final class Foreign extends Base {
    private int localCalls;

    // This declaration does not override Base.token(): it is in another package.
    int token() {
        localCalls++;
        return 22 + localCalls;
    }

    public int throughForeign() {
        return token();
    }
}
