package abs.integration;

public abstract class Shape {
    public abstract double area();
    public abstract double perimeter();
}

public class Square extends Shape {
    double side;
    public Square(double side) { this.side = side; }
    public double area() { return side * side; }
    public double perimeter() { return 4 * side; }
}

public class Circle extends Shape {
    double radius;
    public Circle(double radius) { this.radius = radius; }
    public double area() { return Math.PI * radius * radius; }
    public double perimeter() { return 2 * Math.PI * radius; }
}
