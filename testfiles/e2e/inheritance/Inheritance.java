class Shape {
    protected String name;

    Shape(String name) {
        this.name = name;
    }

    double area() {
        return 0.0;
    }

    String describe() {
        return name + " area=" + area();
    }
}

class Rectangle extends Shape {
    private double w;
    private double h;

    Rectangle(double w, double h) {
        super("rectangle");
        this.w = w;
        this.h = h;
    }

    @Override
    double area() {
        return w * h;
    }
}

class Square extends Rectangle {
    Square(double side) {
        super(side, side);
    }

    @Override
    String describe() {
        return "square " + super.describe();
    }
}

public class Inheritance {
    public static void main(String[] args) {
        Shape[] shapes = new Shape[] {
            new Rectangle(2.0, 3.0),
            new Square(4.0)
        };
        for (Shape s : shapes) {
            System.out.println(s.describe());
        }
    }
}
