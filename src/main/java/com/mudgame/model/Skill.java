package com.mudgame.model;

import java.util.Map;

public class Skill extends Models {
    private String id;
    private String name;
    private String description;
    private int rarity; // 1-7
    private String type; // "BASIC" or "SPECIAL"
    private Map<String, Double> bonuses; // e.g., "strength_growth": 0.1

    public String getId(){
        return id;
    }
    public String getName(){
        return name;
    }
    public String getDescription() { return description; }
    public int getRarity() { return rarity; }
    public String getType() { return type; }
    public Map<String, Double> getBonuses() { return bonuses; }

    public void setId(String id){
        this.id=id;
    }
    public void setName(String name){
        this.name=name;
    }
    public void setDescription(String description) { this.description = description; }
    public void setRarity(int rarity) { this.rarity = rarity; }
    public void setType(String type) { this.type = type; }
    public void setBonuses(Map<String, Double> bonuses) { this.bonuses = bonuses; }
}
