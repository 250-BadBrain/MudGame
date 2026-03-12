package com.mudgame.model;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Room extends Models {
    private String id;
    private String name;
    private String description;
    private Map<String, String> exits; // direction -> roomId
    private List<String> npcs; 
    private List<String> playersInRoom;

    public Room(String id, String name, String description, Map<String, String> exits, List<String> npcs, List<String> playersInRoom) {
        this.id = id;
        this.name = name;
        this.description = description;
        this.exits = exits;
        this.npcs = npcs;
        this.playersInRoom = playersInRoom;
    }

    public Room() {
        playersInRoom = new ArrayList<>();
        npcs = new ArrayList<>();
    }

    public String getId() {
        return id;
    }
    public String getName() {
        return name;
    }
    public String getDescription() {
        return description;
    }
    public Map<String, String> getExits() {
        return exits;
    }
    public List<String> getNpcs() {
        return npcs;
    }
    public List<String> getPlayersInRoom(){
        return playersInRoom;
    }

    public void setId(String Id) {
        this.id=Id;
    }
    public void setName(String Name) {
        this.name=Name;
    }
    public void setDescription(String Description) {
        this.description=Description;
    }
    public void setExits(Map<String, String> exitsMap) {
        this.exits=exitsMap;
    }
    public void setNpcs(List<String> npcsList) {
        this.npcs=npcsList;
    }

    public Room clone(){
        Room copy = new Room();

        copy.id = this.id;
        copy.name = this.name;
        copy.description = this.description;
        if (this.exits != null) copy.exits = new HashMap<>(this.exits);
        if (this.npcs != null) copy.npcs = new ArrayList<>(this.npcs);

        return copy;
    }

    public synchronized void addPlayer(String playerId){
        if (!playersInRoom.contains(playerId)) {
            playersInRoom.add(playerId);
        }
    }
    public synchronized void removePlayer(String playerId){
        playersInRoom.remove(playerId);
    }
    public synchronized void clearPlayer(){
        playersInRoom.clear();
    }
    public synchronized boolean isPlayerInRoom(String playerId){
        if(this.playersInRoom==null){
            return false;
        }
        return playersInRoom.contains(playerId);
    }
}
