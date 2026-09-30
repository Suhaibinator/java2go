package parity.nestedmath;

public class MathFractal {
    public static void main(String[] args) {
        FractalGenerator generator = new FractalGenerator();
        FractalGenerator.Result result = generator.generate(10, 0.5, 0.8);
        System.out.println("Points: " + result.getPoints());
        System.out.println("Area: " + result.getArea());
    }
}
