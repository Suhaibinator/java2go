package parity.physics.simulation;

import parity.physics.model.Particle;
import parity.physics.model.Vector2D;
import java.util.ArrayList;
import java.util.List;

public class PhysicsEngine {
    private List<Particle> particles;
    private Vector2D gravity;

    public PhysicsEngine(Vector2D gravity) {
        this.particles = new ArrayList<>();
        this.gravity = gravity;
    }

    public void addParticle(Particle p) {
        particles.add(p);
    }

    public void step(double dt) {
        for (int i = 0; i < particles.size(); i++) {
            Particle p = particles.get(i);

            // Apply gravity force (F = mg)
            Vector2D gravityForce = new Vector2D(gravity.getX() * p.getMass(), gravity.getY() * p.getMass());
            p.applyForce(gravityForce, dt);

            // Update position
            p.update(dt);
        }
    }

    public void printState(int stepNumber) {
        System.out.println("Step " + stepNumber + ":");
        for (int i = 0; i < particles.size(); i++) {
            Particle p = particles.get(i);
            Vector2D pos = p.getPosition();
            Vector2D vel = p.getVelocity();
            String posStr = "(" + pos.getX() + ", " + pos.getY() + ")";
            String velStr = "(" + vel.getX() + ", " + vel.getY() + ")";
            System.out.println("  Particle[" + p.getId() + "] Pos" + posStr + " Vel" + velStr + " Mass=" + p.getMass());
        }
    }
}
