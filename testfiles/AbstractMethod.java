abstract class Shape {
    abstract double area();
}

class Square extends Shape {
    double side;
    Square(double side) {
        this.side = side;
    }

    double area() {
        return side * side;
    }
}
