package com.mudgame.network.Auth;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.KeySpec;
import java.util.Base64;

import javax.crypto.SecretKeyFactory;
import javax.crypto.spec.PBEKeySpec;

public class Pbkdf2PasswordHasher implements PasswordHasher {
    static final String PREFIX = "pbkdf2";
    private static final int ITERATIONS = 210000;
    private static final int SALT_LENGTH = 16;
    private static final int KEY_LENGTH = 256;
    private static final SecureRandom SECURE_RANDOM = new SecureRandom();

    @Override
    public String hash(String rawPassword) {
        byte[] salt = new byte[SALT_LENGTH];
        SECURE_RANDOM.nextBytes(salt);
        byte[] hash = derive(rawPassword, salt, ITERATIONS);
        return String.join("$",
            PREFIX,
            Integer.toString(ITERATIONS),
            Base64.getUrlEncoder().withoutPadding().encodeToString(salt),
            Base64.getUrlEncoder().withoutPadding().encodeToString(hash));
    }

    @Override
    public boolean matches(String rawPassword, String storedHash) {
        ParsedHash parsedHash = parse(storedHash);
        if (parsedHash == null) {
            return false;
        }

        byte[] actualHash = derive(rawPassword, parsedHash.salt(), parsedHash.iterations());
        return MessageDigest.isEqual(actualHash, parsedHash.hash());
    }

    @Override
    public boolean needsRehash(String storedHash) {
        ParsedHash parsedHash = parse(storedHash);
        return parsedHash == null || parsedHash.iterations() < ITERATIONS;
    }

    private static ParsedHash parse(String storedHash) {
        if (storedHash == null || storedHash.isBlank()) {
            return null;
        }

        String[] parts = storedHash.split("\\$");
        if (parts.length != 4 || !PREFIX.equals(parts[0])) {
            return null;
        }

        try {
            int iterations = Integer.parseInt(parts[1]);
            byte[] salt = Base64.getUrlDecoder().decode(parts[2]);
            byte[] hash = Base64.getUrlDecoder().decode(parts[3]);
            return new ParsedHash(iterations, salt, hash);
        } catch (IllegalArgumentException e) {
            return null;
        }
    }

    private static byte[] derive(String rawPassword, byte[] salt, int iterations) {
        try {
            SecretKeyFactory factory = SecretKeyFactory.getInstance("PBKDF2WithHmacSHA256");
            KeySpec spec = new PBEKeySpec(rawPassword.toCharArray(), salt, iterations, KEY_LENGTH);
            return factory.generateSecret(spec).getEncoded();
        } catch (NoSuchAlgorithmException | InvalidKeySpecException e) {
            throw new IllegalStateException("PBKDF2 algorithm not available", e);
        }
    }

    private record ParsedHash(int iterations, byte[] salt, byte[] hash) {
    }
}