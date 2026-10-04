enum ShadowEnum {
    ONE;
    static int score;
    static int apply(int score) {
        ShadowEnum.score = score;
        return ++ShadowEnum.score + score;
    }
}
public class CampaignStaticEnum {
    public static String run() {
        return ShadowEnum.apply(4) + ":" + ShadowEnum.score;
    }
}
