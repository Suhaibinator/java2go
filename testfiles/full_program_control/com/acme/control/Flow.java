package com.acme.control;

public class Flow {
    public static int compute(int n) {
        int total = 0;

        for (int i = 0; i < n; i++) {
            if (i % 2 == 0) {
                total += i;
            } else {
                total += 1;
            }
        }

        int j = 0;
        while (j < n) {
            total += j;
            j++;
        }

        do {
            total += 1;
            n--;
        } while (n > 0);

        try {
            total += 2;
        } catch (Exception e) {
            total += 100;
        } finally {
            total += 333;
        }

        return total;
    }
}
