package com.mudgame.storage;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.User;
import com.mudgame.utils.Logger;

public class UserStorage {

    private static final Path USERS_DIR;

    static {
        if (new File("server/data").exists()) {
            USERS_DIR = Path.of("server/data/servers/server1/users").toAbsolutePath().normalize();
        } else {
            USERS_DIR = Path.of("data/servers/server1/users").toAbsolutePath().normalize();
        }
    } 
    private final ObjectMapper objectMapper;

    public UserStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper.copy().enable(SerializationFeature.INDENT_OUTPUT);
    }

    public void saveUser(User user) throws SaveException  {
        try {
            Path filePath = resolveUserPath(user.getUsername());
            Files.createDirectories(USERS_DIR);
            objectMapper.writeValue(filePath.toFile(), user);
            
            Logger.info("storage：用户数据已保存：" + user.getUsername());
        
        } catch (IOException | IllegalArgumentException e) {
            Logger.error("storage：保存用户数据失败："+user.getUsername()+"，原因："+e.getMessage());
            throw new SaveException("无法保存用户数据："+user.getUsername(),e);
        }
    }

    public User loadUser(String username) throws LoadException{
        try {
            Path filePath = resolveUserPath(username);
            File file = filePath.toFile();

            if(!file.exists()){
                throw new LoadException("用户文件不存在：" + filePath);
            }
            
            User user = objectMapper.readValue(file, User.class);
            Logger.info("storage：加载用户成功：" + user.getUsername());
            return user;

        } catch (IOException | IllegalArgumentException e) {
            Logger.error("storage：加载用户失败：" + username + "，原因：" + e.getMessage());
            throw new LoadException("无法加载用户数据：" + username, e);
        }
    }

    public boolean deleteUser(String username) throws SaveException {
        try {
            Path filePath = resolveUserPath(username);
            return Files.deleteIfExists(filePath);
        } catch (IOException | IllegalArgumentException e) {
            throw new SaveException("无法删除用户数据：" + username, e);
        }
    }

    private Path resolveUserPath(String username) {
        if (username == null || username.isBlank()) {
            throw new IllegalArgumentException("用户名不能为空");
        }

        Path filePath = USERS_DIR.resolve(username + ".json").normalize();
        if (!filePath.startsWith(USERS_DIR)) {
            throw new IllegalArgumentException("非法用户名路径");
        }

        return filePath;
    }
}