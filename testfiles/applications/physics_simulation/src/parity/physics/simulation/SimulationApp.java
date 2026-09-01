package parity.physics.simulation;

import parity.physics.model.Particle;
import parity.physics.model.Vector2D;

public class SimulationApp {
    public static void main(String[] args) {
        Vector2D gravity = new Vector2D(0, -9.81);
        PhysicsEngine engine = new PhysicsEngine(gravity);

        engine.addParticle(new Particle("P1", new Vector2D(0, 100), new Vector2D(5, 0), 2.0));
        engine.addParticle(new Particle("P2", new Vector2D(10, 50), new Vector2D(-2, 10), 1.5));

        engine.printState(0);

        for (int i = 1; i <= 5; i++) {
            engine.step(0.5); // 0.5 seconds per step
            engine.printState(i);
        }
    }
}
