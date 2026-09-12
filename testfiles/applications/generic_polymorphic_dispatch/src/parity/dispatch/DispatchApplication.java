package parity.dispatch;

import parity.dispatch.basic.GenericDispatchApp;
import parity.dispatch.timing.GenericDispatchTimingApp;
import parity.dispatch.bounded.BoundedDispatchApp;
import parity.dispatch.covariant.CovariantDispatchApp;
import parity.dispatch.factory.FactoryDispatchApp;
import parity.dispatch.shadowing.ShadowedBoundsApp;
import parity.dispatch.targetview.GenericTargetProbe;

public class DispatchApplication {
    public static void main(String[] args) {
        System.out.println(GenericDispatchApp.run());
        System.out.println(GenericDispatchTimingApp.run());
        System.out.println(BoundedDispatchApp.run());
        System.out.println(CovariantDispatchApp.run());
        System.out.println(FactoryDispatchApp.run());
        System.out.println(GenericTargetProbe.run());
        System.out.println(ShadowedBoundsApp.run());
    }
}
