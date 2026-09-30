package parity.physics.model;

public class Particle {
    private String id;
    private Vector2D position;
    private Vector2D velocity;
    private double mass;

    public Particle(String id, Vector2D position, Vector2D velocity, double mass) {
        this.id = id;
        this.position = position;
        this.velocity = velocity;
        this.mass = mass;
    }

    public String getId() { return id; }
    public Vector2D getPosition() { return position; }
    public Vector2D getVelocity() { return velocity; }
    public double getMass() { return mass; }

    public void applyForce(Vector2D force, double dt) {
        // F = ma -> a = F/m
        // v = v0 + at
        Vector2D acceleration = new Vector2D(force.getX() / mass, force.getY() / mass);
        acceleration.multiply(dt);
        this.velocity.add(acceleration);
    }

    public void update(double dt) {
        // p = p0 + vt
        Vector2D deltaP = new Vector2D(velocity.getX(), velocity.getY());
        deltaP.multiply(dt);
        this.position.add(deltaP);
    }
}
