package com.mudgame.model;

import java.util.List;

public class Npc extends Models {
    private String id;
    private String name;
    private String title;
    private String roomId;
    private String description;
    private Attributes attributes;
    private List<Item> items;
    private List<String> shopItems;
    private java.util.Map<String, Integer> stock = new java.util.concurrent.ConcurrentHashMap<>();
    private java.util.Map<String, Integer> skills; // SkillID -> Level

    public Npc() {

    }

    public int getStock(String itemId, int defaultLimit) {
        return stock.computeIfAbsent(itemId, k -> defaultLimit);
    }

    public void reduceStock(String itemId, int amount) {
        stock.computeIfPresent(itemId, (k, v) -> v == -1 ? -1 : Math.max(0, v - amount));
    }

    public Npc(String id, String name ,String roomId, Attributes attributes, List<Item> items, java.util.Map<String, Integer> skills) {
        this.id = id;
        this.name = name;
        this.roomId = roomId;
        this.attributes = attributes;
        this.items = items;
        this.skills = skills;
    }

    public String getId() {
        return id;
    }
    public String getName() {
        return name;
    }
    public String getDescription(){
        return description;
    }
    public String getRoomId() {
        return roomId;
    }
    public Attributes getAttributes(){
        return attributes;
    }
    public List<Item> getItems() {
        return items;
    }
    public java.util.Map<String, Integer> getSkills() {
        return skills;
    }
    public String getTitle() {
        return title;
    }
    public void setTitle(String title) {
        this.title = title;
    }
    public List<String> getShopItems() {
        return shopItems;
    }
    public void setShopItems(List<String> shopItems) {
        this.shopItems = shopItems;
    }

    public void setId(String Id) {
        this.id=Id;
    }
    public void setName(String name) {
        this.name=name;
    }
    public void setDescription(String description){
        this.description=description;
    }
    public void setRoomId(String RoomId) {
        this.roomId=RoomId;
    }
    public void setAttributes(Attributes attributes){
        this.attributes=attributes;
    }
    public void setItems(List<Item> itemsList) {
        this.items=itemsList;
    }
    public void setSkills(java.util.Map<String, Integer> skills) {
        this.skills = skills;
    }
}
