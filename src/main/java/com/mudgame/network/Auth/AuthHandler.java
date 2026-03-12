package com.mudgame.network.Auth;

import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.storage.UserStorage; 
import com.mudgame.model.User;
import com.mudgame.exception.LoadException; 
import com.mudgame.utils.Logger;

import java.util.Collections; 
import java.util.HashMap;
import java.util.Map;

/** 登录/注册处理。改为：username -> User object (包含 passwordHash, characterIds) */
public class AuthHandler {

    private final UserStorage userStorage; // 新依赖
    private final PasswordHasher hasher;
    private final ReconnectTokenService reconnectTokenService;

        // 更改构造函数签名：只依赖 UserStorage 和 PasswordHasher
        public AuthHandler(UserStorage userStorage, PasswordHasher hasher, ReconnectTokenService reconnectTokenService) {
        this.userStorage = userStorage;
        this.hasher = hasher;
            this.reconnectTokenService = reconnectTokenService;
    }

    public GameResult handleLogin(CommandMessage cmd) {
        String username = safeTrim((String) cmd.getArg("username"));
        String password = safeTrim((String) cmd.getArg("password"));

        if (isBlank(username) || isBlank(password)) {
            return sysFail("login", "missing_credentials");
        }

        User user;
        try {
            // 尝试加载 User 对象
            user = userStorage.loadUser(username);
        } catch (LoadException e) {
            return sysFail("login", "user_not_found");
        } catch (Exception e) {
            Logger.error("[login] load user failed: " + e.getMessage());
            return sysFail("login", "storage_error");
        }
        
        // 验证密码
        String storedHash = user.getPasswordHash();
        if (!hasher.matches(password, storedHash)) {
            return sysFail("login", "wrong_password");
        }

        String reconnectToken = null;
        try {
            if (hasher.needsRehash(storedHash)) {
                user.setPasswordHash(hasher.hash(password));
            }
            user.setLastLogin(System.currentTimeMillis());
            reconnectToken = reconnectTokenService.issueToken(user);
            userStorage.saveUser(user);
        } catch (Exception e) {
            Logger.warn("[login] save lastLogin/token failed for user " + username + ": " + e.getMessage());
            reconnectToken = null;
        }
        
        // 登录成功，返回 User ID 和角色列表
        Map<String,Object> results = new HashMap<>();
        results.put("userId", user.getUsername()); // 使用 username 作为 User ID
        results.put("characterIds", user.getCharacterIds()); // 返回角色ID列表
        if (reconnectToken != null) {
            results.put("reconnectToken", reconnectToken);
        }
        
        return sysOk("login", results);
    }

    public GameResult handleRegister(CommandMessage cmd) {
        String username = safeTrim((String) cmd.getArg("username"));
        String password = safeTrim((String) cmd.getArg("password"));

        if (isBlank(username) || isBlank(password)) {
            return sysFail("register", "missing_credentials");
        }
        if (username.length() < 3 || username.length() > 20) {
            return sysFail("register", "bad_username");
        }
        if (password.length() < 6) {
            return sysFail("register", "weak_password");
        }
        
        // 检查用户名是否已存在
        try {
            userStorage.loadUser(username);
            return sysFail("register", "username_taken"); 
        } catch (LoadException ignored) {
            // 预期：加载失败，继续注册流程
        } catch (Exception e) {
            Logger.error("[register] load user failed: " + e.getMessage());
            return sysFail("register", "storage_error");
        }

        // 创建新的 User 对象，角色列表为空
        String hash = hasher.hash(password);
        User newUser = new User(username, hash);
        newUser.setLastLogin(System.currentTimeMillis());
        String reconnectToken = reconnectTokenService.issueToken(newUser);
        
        try {
            // 保存 User 对象
            userStorage.saveUser(newUser); 
        } catch (Exception e) {
            Logger.error("[register] save user failed: " + e.getMessage());
            return sysFail("register", "save_user_failed");
        }
        
        // 注册成功，返回 User ID 和空的 characterIds
        Map<String,Object> results = new HashMap<>();
        results.put("userId", username); 
        results.put("characterIds", Collections.emptyList()); 
        results.put("reconnectToken", reconnectToken);
        return sysOk("register", results);
    }

    private static boolean isBlank(String s){ return s == null || s.trim().isEmpty(); }
    private static String safeTrim(String s){ return s == null ? null : s.trim(); }

    private static GameResult sysOk(String subType, Map<String,Object> results) {
        return new GameResult("system", subType, true, System.currentTimeMillis(), results, null);
    }
    private static GameResult sysFail(String subType, String error) {
        return new GameResult("system", subType, false, System.currentTimeMillis(),
                Map.of("error", error), null);
    }
}