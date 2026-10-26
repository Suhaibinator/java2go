package campaign.probe.app;

import campaign.probe.base.Base;
import campaign.probe.base.Local;
import campaign.probe.foreign.Foreign;

public final class Probe {
    public static void main(String[] args) {
        Foreign foreign = new Foreign();
        Base asBase = foreign;
        Local local = new Local();
        int[] got = {
                asBase.throughBase(),
                foreign.throughForeign(),
                asBase.throughCallback(foreign),
                foreign.throughForeign(),
                local.throughBase(),
                asBase.throughCallback(local)
        };
        int[] want = {12, 23, 13, 24, 33, 33};
        System.out.println("observed=" + got[0] + "," + got[1] + "," + got[2]
                + "," + got[3] + "," + got[4] + "," + got[5]);
        for (int i = 0; i < got.length; i++) {
            if (got[i] != want[i]) {
                throw new AssertionError("case " + i + ": " + got[i] + " != " + want[i]);
            }
        }
        System.out.println("base=" + got[0] + "," + got[2]);
        System.out.println("foreign=" + got[1] + "," + got[3]);
        System.out.println("local=" + got[4] + "," + got[5]);
    }
}
