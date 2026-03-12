package com.mudgame.model;

import java.util.List;
import java.util.Map;
import java.util.HashMap;
import java.util.concurrent.locks.ReentrantLock;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class Player extends Models {
    private transient final ReentrantLock lock = new ReentrantLock();
    private String id;
    private String characterName;
    private String userId;
    private String roomId;
    private String title;
    private List<String> unlockedTitles;
    private long money;
    private Attributes attributes;
    private Map<String, Integer> items = new HashMap<>();
    private Map<String, String> equipment; // Slot -> ItemId
    private Map<String, Long> skills; // SkillID -> Total Potential Consumed
    private PlayerState state = PlayerState.IDLE;
    private String targetId;
    private String learningSkillId;
    private String combatType; // "KILL" or "SPAR"

    public Player(){

    }
    public Player(String id, String characterName, String userId, String roomId, Attributes attributes, List<String> items, Map<String, Long> skills) {
        this.id = id;
        this.characterName = characterName;
        this.userId = userId;
        this.roomId = roomId;
        this.attributes = attributes;
        this.setItems(items); // Use setter for conversion
        this.equipment = new HashMap<>();
        this.skills = skills != null ? skills : new HashMap<>();
        this.money = 10000; // Default 1 Gold
        this.title = "";
    }


    public String getId() {
        return id;
    }
    public String getCharacterName() {
        return characterName;
    }
    public String getUserId() {
        return userId;
    }
    public String getRoomId() {
        return roomId;
    }
    public String getTitle() {
        return title;
    }
    public List<String> getUnlockedTitles() {
        return unlockedTitles;
    }
    public long getMoney() {
        return money;
    }
    public Attributes getAttributes(){
        if (attributes != null) {
            updateSkillBonuses(); // Ensure bonuses are up to date
            attributes.recalculateDerivedAttributes();
        }
        return attributes;
    }
    public Map<String, Integer> getItems() {
        return items;
    }
    public Map<String, String> getEquipment() {
        return equipment;
    }
    public Map<String, Long> getSkills() {
        return skills;
    }
    public PlayerState getState() {
        return state;
    }
    public String getTargetId() {
        return targetId;
    }
    public String getLearningSkillId() {
        return learningSkillId;
    }
    public String getCombatType() {
        return combatType;
    }

    public void setId(String Id) {
        this.id=Id;
    }
    public void setCharacterName(String characterName) {
        this.characterName = characterName;
    }
    public void setUserId(String userId) {
        this.userId = userId;
    }
    public void setRoomId(String RoomId) {
        this.roomId=RoomId;
    }
    public void setAttributes(Attributes attributes){
        this.attributes=attributes;
    }
    @SuppressWarnings("unchecked")
    public void setItems(Object itemsObj) {
        if (itemsObj instanceof List) {
            List<String> list = (List<String>) itemsObj;
            this.items = new HashMap<>();
            for (String id : list) {
                this.items.merge(id, 1, (a, b) -> a + b);
            }
        } else if (itemsObj instanceof Map) {
            this.items = (Map<String, Integer>) itemsObj;
        } else {
            this.items = new HashMap<>();
        }
    }
    public void setEquipment(Map<String, String> equipment) {
        this.equipment = equipment;
    }
    @SuppressWarnings("unchecked")
    public void setSkills(Object skillsObj) {
        if (skillsObj instanceof List) {
            List<String> list = (List<String>) skillsObj;
            this.skills = new HashMap<>();
            for (String id : list) {
                this.skills.put(id, 0L); // Default 0 potential for old skills
            }
        } else if (skillsObj instanceof Map) {
            // JSON deserialization might give Map<String, Integer> or Map<String, Long>
            // We need to ensure values are Long
            Map<?, ?> map = (Map<?, ?>) skillsObj;
            this.skills = new HashMap<>();
            for (Map.Entry<?, ?> entry : map.entrySet()) {
                if (entry.getKey() instanceof String && entry.getValue() instanceof Number) {
                    this.skills.put((String) entry.getKey(), ((Number) entry.getValue()).longValue());
                }
            }
        } else {
            this.skills = new HashMap<>();
        }
    }

    public int getSkillLevel(String skillId) {
        if (skills == null || !skills.containsKey(skillId)) {
            return 0;
        }
        long potential = skills.get(skillId);
        if (potential < 200) return 0;
        
        // Binary search for level
        // Cost(L) = 0.02 * L^1.6 + 200
        // Total(L) = Sum(Cost(i) for i=0 to L-1)
        // We want to find L such that Total(L) <= potential < Total(L+1)
        
        int low = 0;
        int high = 5000; // Reasonable max level cap for search
        int level = 0;
        
        while (low <= high) {
            int mid = low + (high - low) / 2;
            long pMid = calculateTotalPotentialForLevel(mid);
            
            if (pMid <= potential) {
                level = mid;
                low = mid + 1;
            } else {
                high = mid - 1;
            }
        }
        return level;
    }

    public double getSkillPercentage(String skillId) {
        if (skills == null || !skills.containsKey(skillId)) {
            return 0.0;
        }
        long potential = skills.get(skillId);
        if (potential < 200) return 0.0;
        
        int currentLevel = getSkillLevel(skillId);
        long pCurrent = calculateTotalPotentialForLevel(currentLevel);
        long pNext = calculateTotalPotentialForLevel(currentLevel + 1);
        
        if (pNext == pCurrent) return 100.0; // Should not happen
        
        double progress = (double)(potential - pCurrent) / (double)(pNext - pCurrent);
        return Math.max(0.0, Math.min(1.0, progress)) * 100.0;
    }
    
    // Helper to calculate total potential required to reach level L
    // Total(L) = Sum(Cost(i) for i=0 to L-1)
    // Cost(i) = 0.02 * i^1.6 + 200
    private long calculateTotalPotentialForLevel(int level) {
        if (level <= 0) return 0;
        // Since we don't have a closed form sum, and max level is not huge, we can approximate or iterate.
        // But iterating 5000 times for every check is slow if called frequently.
        // However, binary search calls this log(5000) ~ 13 times.
        // Inside, if we iterate 2500 times on average... 13 * 2500 = 32500 ops.
        // That's fine for a single call.
        // Optimization: Cache or use approximation.
        // Approximation: Integral(0.02 * x^1.6 + 200) dx = 0.02 * x^2.6 / 2.6 + 200x
        // Let's use the integral approximation for speed, it should be close enough for a game.
        // Or better, pre-calculate a table if levels are discrete.
        // Given the constraints, let's use the integral approximation which is O(1).
        // Total(L) ≈ 0.0076923 * L^2.6 + 200 * L
        
        double term1 = (0.02 / 2.6) * Math.pow(level, 2.6);
        double term2 = 200.0 * level;
        return (long) (term1 + term2);
    }
    
    public void updateSkillBonuses() {
        if (attributes == null) return;
        
        int bonusStr = 0;
        int bonusAgi = 0;
        int bonusParry = 0;
        int bonusMp = 0;
        
        // Basic Fist: +1 Post-natal Strength per 10 levels
        int fistLevel = getSkillLevel("basic_fist");
        bonusStr += fistLevel / 10;
        
        // Basic Parry: +1 Post-natal Parry per 10 levels
        int parryLevel = getSkillLevel("basic_parry");
        bonusParry += parryLevel / 10; // Assuming flat parry bonus
        
        // Basic Internal: +10 Max Internal Force per 1 level
        int internalLevel = getSkillLevel("basic_internal");
        bonusMp += internalLevel * 10;
        
        // Basic Internal: Convert 30% of Trained Max MP to Max HP if level > 0
        if (internalLevel > 0) {
            attributes.setSkillMpToHpRatio(0.3);
        } else {
            attributes.setSkillMpToHpRatio(0.0);
        }
        
        // Basic Light: +1 Post-natal Agility per 10 levels
        int lightLevel = getSkillLevel("basic_light");
        bonusAgi += lightLevel / 10;
        
        attributes.setSkillPostStrength(bonusStr);
        attributes.setSkillPostAgility(bonusAgi);
        attributes.setSkillPostParry(bonusParry);
        attributes.setSkillMaxMpBonus(bonusMp);
        
        attributes.recalculateDerivedAttributes();
    }
    public void setTargetId(String targetId) {
        this.targetId = targetId;
    }
    public void setLearningSkillId(String learningSkillId) {
        this.learningSkillId = learningSkillId;
    }
    public void setCombatType(String combatType) {
        this.combatType = combatType;
    }
    public void setState(PlayerState state) {
        this.state = state;
    }
    public void setTitle(String title) {
        this.title = title;
    }
    public void setUnlockedTitles(List<String> unlockedTitles) {
        this.unlockedTitles = unlockedTitles;
    }
    public void setMoney(long money) {
        this.money = money;
    }

    @JsonIgnore
    public ReentrantLock getLock() {
        return lock;
    }
}