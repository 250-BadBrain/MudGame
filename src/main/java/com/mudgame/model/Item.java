package com.mudgame.model;

public class Item extends Models {
    private String id;
    private String name;
    private String description;
    private boolean stackable;
    private boolean sellable;
    private int value;
    private int rarity;
    private String unit;
    private String equipmentSlot;
    private int limit = -1; // -1 means unlimited

    public Item() {

    }

    public Item(String id, String name, String description, boolean stackable, boolean sellable, int value,
            int rarity, String unit, String equipmentSlot, int limit) {
        this.id = id;
        this.name = name;
        this.description = description;
        this.stackable = stackable;
        this.sellable = sellable;
        this.value = value;
        this.rarity = rarity;
        this.unit = unit;
        this.equipmentSlot = equipmentSlot;
        this.limit = limit;
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

    public boolean isStackable() {
        return stackable;
    }

    public boolean isSellable() {
        return sellable;
    }

    public int getValue() {
        return value;
    }

    public int getRarity() {
        return rarity;
    }

    public String getUnit() {
        return unit;
    }

    public String getEquipmentSlot() {
        return equipmentSlot;
    }

    public int getLimit() {
        return limit;
    }

    public void setId(String id) {
        this.id = id;
    }

    public void setName(String name) {
        this.name = name;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public void setStackable(boolean stackable) {
        this.stackable = stackable;
    }

    public void setSellable(boolean sellable) {
        this.sellable = sellable;
    }

    public void setValue(int value) {
        this.value = value;
    }

    public void setRarity(int rarity) {
        this.rarity = rarity;
    }

    public void setUnit(String unit) {
        this.unit = unit;
    }

    public void setEquipmentSlot(String equipmentSlot) {
        this.equipmentSlot = equipmentSlot;
    }

    public void setLimit(int limit) {
        this.limit = limit;
    }
}
