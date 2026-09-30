// seed: 285
public class Gen {
    public static void main(String[] args) {
        int[] arr0 = {388, 1000000, 2147483647, 255};
        arr0[0] = arr0[arr0.length - 1];
        int i1 = 0;
        for (int q2 = 0; q2 < arr0.length; q2++) { i1 += arr0[q2]; }
        StringBuilder sb4 = new StringBuilder();
        sb4.append(2147483647);
        String s3 = sb4.toString();
        StringBuilder sb6 = new StringBuilder();
        sb6.append(i1);
        sb6.append(false);
        sb6.append("gamma");
        String s5 = sb6.toString();
        int i7 = 0;
        while (i7 < 4) {
            i1 += i1;
            s3 += "gamma" + 64;
            i7++;
        }
        int i8 = 0;
        while (i8 < 1) {
            i7 -= i8;
            i8++;
        }
        int[] arr9 = {2147483647, i7, 65535};
        arr9[0] = arr9[arr9.length - 1];
        int i10 = 0;
        for (int q11 = 0; q11 < arr9.length; q11++) { i10 += arr9[q11]; }
        boolean b12 = (((i1 % (i7 | 1)) % (i7 | 1)) % (i1 | 1)) >= i7;
        String s13 = "";
        int i14 = s13.length();
        String s15 = s13.toUpperCase();
        int i16 = 0;
        while (i16 < 2) {
            i10 -= i8;
            i16++;
        }
        i8--;
        System.out.println(i1);
        System.out.println(i7);
        System.out.println(i8);
        System.out.println(i10);
        System.out.println(i14);
        System.out.println(i16);
        System.out.println(b12);
        System.out.println(s3);
        System.out.println(s5);
        System.out.println(s13);
        System.out.println(s15);
    }
}
