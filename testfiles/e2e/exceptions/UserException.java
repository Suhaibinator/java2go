class AppException extends RuntimeException {
    AppException(String message) {
        super(message);
    }
}

class NotFoundException extends AppException {
    NotFoundException(String message) {
        super(message);
    }
}

public class UserException {
    static String lookup(int id) {
        try {
            if (id < 0) {
                throw new NotFoundException("no id " + id);
            }
            if (id == 0) {
                throw new AppException("zero id");
            }
            return "found " + id;
        } catch (NotFoundException e) {
            return "not-found: " + e.getMessage();
        } catch (AppException e) {
            return "app: " + e.getMessage();
        }
    }

    static String lookupBySupertype(int id) {
        try {
            if (id < 0) {
                throw new NotFoundException("missing " + id);
            }
            return "ok " + id;
        } catch (RuntimeException e) {
            return "caught-as-runtime: " + e.getMessage();
        }
    }

    public static void main(String[] args) {
        System.out.println(lookup(5));
        System.out.println(lookup(0));
        System.out.println(lookup(-1));
        System.out.println(lookupBySupertype(-7));
        System.out.println(lookupBySupertype(3));
    }
}
