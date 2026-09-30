package com.acme.app;

import com.acme.services.UserService;

public class App {
    public static String run() {
        UserService svc = new UserService();
        return svc.publicName();
    }
}
