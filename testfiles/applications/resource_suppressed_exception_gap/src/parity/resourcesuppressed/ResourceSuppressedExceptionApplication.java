package parity.resourcesuppressed;

class ThrowingResource implements AutoCloseable {
    @Override
    public void close() {
        throw new IllegalStateException();
    }
}

public final class ResourceSuppressedExceptionApplication {
    private ResourceSuppressedExceptionApplication() {}

    public static void main(String[] args) {
        try (ThrowingResource resource = new ThrowingResource()) {
            throw new IllegalArgumentException();
        } catch (IllegalArgumentException ex) {
            System.out.println("body");
        } catch (IllegalStateException ex) {
            System.out.println("close");
        }
    }
}
