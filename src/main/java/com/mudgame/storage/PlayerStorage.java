package com.mudgame.storage;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.stream.Stream;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.Player;
import com.mudgame.utils.Logger;

public class PlayerStorage {

    private static final Path PLAYERS_DIR;

    static {
        if (new File("server/data").exists()) {
            PLAYERS_DIR = Path.of("server/data/servers/server1/players").toAbsolutePath().normalize();
        } else {
            PLAYERS_DIR = Path.of("data/servers/server1/players").toAbsolutePath().normalize();
        }
    }
    private final ObjectMapper objectMapper;

    public PlayerStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper.copy().enable(SerializationFeature.INDENT_OUTPUT);
    }

    public void savePlayer(Player player) throws SaveException  {
        try {
            Path filePath = resolvePlayerPath(player.getId());
            Files.createDirectories(PLAYERS_DIR);
            objectMapper.writeValue(filePath.toFile(),player);

            Logger.info("storage：玩家数据已保存：" + player.getCharacterName() + "(ID：" + player.getId() + ")");
        
        } catch (IOException | IllegalArgumentException e) {
            Logger.error("storage：保存玩家数据失败："+player.getId()+"，原因："+e.getMessage());
            throw new SaveException("无法保存玩家数据："+player.getId(),e);
        }
    }

    public Player loadPlayer(String playerId) throws LoadException{
        try {
            Path filePath = resolvePlayerPath(playerId);
            File file = filePath.toFile();

            if(!file.exists()){
                throw new LoadException("玩家文件不存在：" + filePath);
            }
            
            Player player = objectMapper.readValue(file, Player.class);
            Logger.info("storage：加载玩家成功：" + player.getCharacterName() + "(ID：" + player.getId() + ")");
            return player;

        } catch (IOException | IllegalArgumentException e) {
            Logger.error("storage：加载玩家失败：" + playerId + "，原因：" + e.getMessage());
            throw new LoadException("无法加载玩家数据：" + playerId, e);
        }
    }

    public boolean characterNameExists(String characterName) {
        // 忽略大小写
        final String targetName = characterName.trim().toLowerCase(); 
        File playerDir = PLAYERS_DIR.toFile();

        if (!playerDir.exists() || !playerDir.isDirectory()) {
            return false;
        }

        try (Stream<Path> paths = Files.walk(playerDir.toPath(), 1)) {
            return paths
                .filter(Files::isRegularFile)
                .anyMatch(path -> {
                    try {
                        if (path.toString().endsWith(".json")) {
                            Player player = objectMapper.readValue(path.toFile(), Player.class);
                            return player.getCharacterName() != null && 
                                   player.getCharacterName().trim().toLowerCase().equals(targetName);
                        }
                    } catch (IOException e) {
                        Logger.error("storage：检查角色名时加载文件失败：" + path.getFileName() + "，原因：" + e.getMessage());
                        // 忽略错误文件，继续检查
                    }
                    return false;
                });
        } catch (IOException e) {
            Logger.error("storage：遍历玩家目录失败：" + e.getMessage());
            return true; // 为了安全起见，如果遍历失败，暂时认为存在，防止冲突
        }
    }

    public java.util.List<String> getAllPlayerIds() {
        File playerDir = PLAYERS_DIR.toFile();
        if (!playerDir.exists() || !playerDir.isDirectory()) {
            return java.util.Collections.emptyList();
        }
        
        java.util.List<String> ids = new java.util.ArrayList<>();
        File[] files = playerDir.listFiles((dir, name) -> name.endsWith(".json"));
        if (files != null) {
            for (File file : files) {
                String filename = file.getName();
                ids.add(filename.substring(0, filename.length() - 5));
            }
        }
        return ids;
    }

    public boolean deletePlayer(String playerId) throws SaveException {
        try {
            Path filePath = resolvePlayerPath(playerId);
            return Files.deleteIfExists(filePath);
        } catch (IOException | IllegalArgumentException e) {
            throw new SaveException("无法删除玩家数据：" + playerId, e);
        }
    }

    private Path resolvePlayerPath(String playerId) {
        if (playerId == null || playerId.isBlank()) {
            throw new IllegalArgumentException("角色ID不能为空");
        }

        Path filePath = PLAYERS_DIR.resolve(playerId + ".json").normalize();
        if (!filePath.startsWith(PLAYERS_DIR)) {
            throw new IllegalArgumentException("非法角色路径");
        }

        return filePath;
    }
}
