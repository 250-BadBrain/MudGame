package com.mudgame.model;

import java.util.HashMap;
import java.util.Map;

public class World extends Models {
    private String id;      //luoyang
    private String name;    //洛阳城
    private String description; // 世界描述
    private Map<String, Room> rooms = new HashMap<>();

    public String getId(){
        return id;
    }
    public String getName(){
        return name;
    }
    public String getDescription() {
        return description;
    }
    public Map<String,Room> getRooms(){
        return rooms;
    }
    public Room getRoom(String roomId){
        return rooms.get(roomId);
    }

    public void setId(String id){
        this.id=id;
    }
    public void setName(String name){
        this.name=name;
    }
    public void setDescription(String description) {
        this.description = description;
    }
    public void setRooms(Map<String,Room>rooms){
        this.rooms=rooms;
    }
}