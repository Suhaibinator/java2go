package parity.nestedmath;

public class FractalGenerator {
    private int iterations;

    public Result generate(int iterations, double startX, double startY) {
        this.iterations = iterations;
        Point start = new Point(startX, startY);

        Processor processor = new Processor() {
            @Override
            public double processX(double x, double y) {
                return Math.sin(x * Math.PI) - Math.cos(y * Math.PI);
            }
            @Override
            public double processY(double x, double y) {
                return Math.cos(x * Math.PI) + Math.sin(y * Math.PI);
            }
        };

        return compute(start, processor);
    }

    private Result compute(Point start, Processor processor) {
        double currentX = start.x;
        double currentY = start.y;
        double totalDist = 0;

        for (int i = 0; i < iterations; i++) {
            double nextX = processor.processX(currentX, currentY);
            double nextY = processor.processY(currentX, currentY);
            totalDist += Math.sqrt(Math.pow(nextX - currentX, 2) + Math.pow(nextY - currentY, 2));
            currentX = nextX;
            currentY = nextY;
        }

        return new Result(iterations, totalDist);
    }

    private class Point {
        double x, y;
        Point(double x, double y) {
            this.x = x;
            this.y = y;
        }
    }

    interface Processor {
        double processX(double x, double y);
        double processY(double x, double y);
    }

    public static class Result {
        private int points;
        private double area;

        public Result(int points, double area) {
            this.points = points;
            this.area = area;
        }

        public int getPoints() { return points; }
        public double getArea() { return area; }
    }
}
