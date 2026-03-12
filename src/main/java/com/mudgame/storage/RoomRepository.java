package com.mudgame.storage;

import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.Room;
import com.mudgame.utils.Logger;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.JsonNode;

import java.io.InputStream;
import java.io.File;
import java.io.IOException;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

public class RoomRepository {
    
    private static Map<String,Room> STATIC_ROOMS=new HashMap<>();

    private static final String STATIC_DIR="worlds/";
    private static final String DYNAMIC_DIR;

    static {
        if (new File("server/data").exists()) {
            DYNAMIC_DIR = "server/data/servers/server1/rooms/";
        } else {
            DYNAMIC_DIR = "data/servers/server1/rooms/";
        }
    }
    private static ObjectMapper objectMapper;

    public RoomRepository(ObjectMapper objectMapper) {
        RoomRepository.objectMapper = objectMapper;
    }

    public static void loadStaticRooms(String worldId) throws LoadException{
        String path = STATIC_DIR + worldId + ".json";
        InputStream tempIs = RoomRepository.class.getClassLoader().getResourceAsStream(path);
        if (tempIs == null) {
            tempIs = RoomRepository.class.getResourceAsStream("/" + path);
        }
        
        if (tempIs == null) {
             Logger.error("storage：未找到世界文件：" + path);
             return;
        }

        try(InputStream is = tempIs){
            JsonNode root=objectMapper.readTree(is);
            JsonNode roomsNode=root.get("rooms");
            
            if(roomsNode!=null&&roomsNode.isObject()){
                roomsNode.fields().forEachRemaining(entry->{
                    String roomId=entry.getKey();
                    
                    Room room=parseRoom(roomId,entry.getValue());
                    STATIC_ROOMS.put(roomId,room);
                });
            }
            Logger.info("storage：加载静态世界 [" + worldId + "]完成。");
        }catch(Exception e){
            Logger.error("storage：加载静态房间失败："+e.getMessage());
        }
    }

    public static Room getStaticRoom(String roomId){
        return STATIC_ROOMS.get(roomId);
    }

    public static Set<String> getRoomIds(){
        return Collections.unmodifiableSet(STATIC_ROOMS.keySet());
    }

    public Room loadDynamicRoom(String roomId) throws LoadException{
        Room room=new Room();

        File file=new File(DYNAMIC_DIR+roomId+".json");
        if(!file.exists()){
            Logger.error("storage：加载动态房间["+roomId+"]失败。");
            return room;
        }

        try{
            JsonNode node=objectMapper.readTree(file);
            if(node.has("players")){
                node.get("players").forEach(p->room.getPlayersInRoom().add(p.asText()));
            }
            if(node.has("npcs")){
                node.get("npcs").forEach(n->room.getNpcs().add(n.asText()));
            }
            Logger.info("storage：加载动态房间["+roomId+"]成功！");
        }catch(IOException e){
            Logger.info("storage：加载动态房间["+roomId+"]失败，原因："+e.getMessage());
            throw new LoadException("加载动态房间失败："+roomId,e);
        }
        return room;
    }

    public void saveDynamicRoom(Room room) throws SaveException {
        File file=new File(DYNAMIC_DIR+room.getId()+".json");
        file.getParentFile().mkdirs();

        Map<String,Object> data = new HashMap<>();
        data.put("players",room.getPlayersInRoom());
        data.put("npcs",room.getNpcs());
        try{
            objectMapper.writerWithDefaultPrettyPrinter().writeValue(file,data);
            Logger.info("storage：存储动态房间["+room.getId()+"]成功！");
        }catch(IOException e){
            Logger.info("storage：存储动态房间["+room.getId()+"]失败，原因："+e.getMessage());
            throw new SaveException("存储动态房间失败："+room.getId(),e);
        }
    }

    private static Room parseRoom(String roomId,JsonNode node){
        Room room=new Room();
        room.setId(roomId);
        if(node.has("name")){
            room.setName(node.get("name").asText());
        }
        if(node.has("description")){
            room.setDescription(node.get("description").asText());
        }
        if(node.has("exits")){
            Map<String,String>exits=new HashMap<>();
            JsonNode exitsNode=node.get("exits");
            if(exitsNode.isObject()){
                exitsNode.fields().forEachRemaining(entry->{
                    String dir=entry.getKey();
                    String targetRoomId=entry.getValue().asText();
                    exits.put(dir,targetRoomId);
                });
            }
            room.setExits(exits);
        }
        if(node.has("npcs")){
            JsonNode npcsNode=node.get("npcs");
            if(npcsNode.isArray()){
                npcsNode.forEach(n->room.getNpcs().add(n.asText()));
            }
        }
        return room;
    }
}