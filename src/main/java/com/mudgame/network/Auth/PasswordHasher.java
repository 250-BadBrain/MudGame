package com.mudgame.network.Auth;

public interface PasswordHasher {
    String hash(String rawPassword);
    boolean matches(String rawPassword, String storedHash);

    default boolean needsRehash(String storedHash) {
        return false;
    }
}

