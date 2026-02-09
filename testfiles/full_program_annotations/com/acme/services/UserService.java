package com.acme.services;

public class UserService {
    @Skip
    String token = "abc";

    @Skip
    public String internalName() {
        return this.token;
    }

    public String publicName() {
        return "user";
    }
}
