package com.mudgame.network.Auth;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.util.Base64;

import com.mudgame.model.User;

public class ReconnectTokenService {
    private static final long TOKEN_TTL_MILLIS = 30L * 24 * 60 * 60 * 1000;
    private static final SecureRandom SECURE_RANDOM = new SecureRandom();

    public String issueToken(User user) {
        String token = generateToken();
        user.setReconnectTokenHash(hashToken(token));
        user.setReconnectTokenExpiresAt(System.currentTimeMillis() + TOKEN_TTL_MILLIS);
        return token;
    }

    public boolean isValid(User user, String reconnectToken) {
        if (user == null || reconnectToken == null || reconnectToken.isBlank()) {
            return false;
        }
        if (user.getReconnectTokenHash() == null || user.getReconnectTokenExpiresAt() < System.currentTimeMillis()) {
            return false;
        }

        String actualHash = hashToken(reconnectToken);
        return MessageDigest.isEqual(
            user.getReconnectTokenHash().getBytes(StandardCharsets.UTF_8),
            actualHash.getBytes(StandardCharsets.UTF_8));
    }

    private static String generateToken() {
        byte[] bytes = new byte[32];
        SECURE_RANDOM.nextBytes(bytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }

    private static String hashToken(String token) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(token.getBytes(StandardCharsets.UTF_8));
            return Base64.getUrlEncoder().withoutPadding().encodeToString(hash);
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException("SHA-256 algorithm not available", e);
        }
    }
}