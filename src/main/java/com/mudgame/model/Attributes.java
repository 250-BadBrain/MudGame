package com.mudgame.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class Attributes {
    private int maxHp;
    private int currentHp;
    private int maxMp; // Internal Force Limit (Theoretical Max)
    private int trainedMaxMp; // Real Max Internal Force (Current Cap)
    private int currentMp;
    private int attack;
    private int defense;
    private int hitRate;       // 命中
    private int dodge;         // 躲闪
    private int parry;         // 招架
    private int attackSpeed;   // 攻击速度
    private int critRate;      // 暴击 (百分比)
    private long experience;   // 经验
    private long potential;    // 潜能
    
    // Innate Attributes
    private int strength;      // 臂力
    private int constitution;  // 根骨
    private int agility;       // 身法
    private int intelligence;  // 悟性
    private int appearance;    // 容貌
    private int luck;          // 福禄

    // Post-natal Attributes
    private int postStrength;
    private int postConstitution;
    private int postAgility;
    private int postIntelligence;

    // Bonuses
    private int flatAttackBonus;
    private double pctAttackBonus = 1.0;
    private int flatHitBonus;
    private double pctHitBonus = 1.0;
    private int flatParryBonus;
    private double pctParryBonus = 1.0;
    private int flatDefenseBonus;
    private double pctDefenseBonus = 1.0;
    private int flatDodgeBonus;
    private double pctDodgeBonus = 1.0;
    
    // Efficiency Bonuses
    private int flatMeditationBonus;
    private double pctMeditationBonus = 1.0;
    private int flatLearningBonus;
    private double pctLearningBonus = 1.0;

    // Skill Bonuses (Calculated from Player Skills)
    private int skillPostStrength;
    private int skillPostAgility;
    private int skillPostParry; // Treated as flat parry bonus
    private int skillMaxMpBonus;
    private double skillMpToHpRatio; // Ratio of Trained Max MP converted to Max HP

    public Attributes(){

    }

    public Attributes(int maxHp,int currentHp,int maxMp,int currentMp,int attack,int defense, int hitRate, int dodge, int parry, int attackSpeed, int critRate, long experience, long potential, int strength, int constitution, int agility, int intelligence, int appearance, int luck){
        this.maxHp=maxHp;
        this.currentHp=currentHp;
        this.maxMp=maxMp;
        this.currentMp=currentMp;
        this.attack=attack;
        this.defense=defense;
        this.hitRate = hitRate;
        this.dodge = dodge;
        this.parry = parry;
        this.attackSpeed = attackSpeed;
        this.critRate = critRate;
        this.experience = experience;
        this.potential = potential;
        this.strength = strength;
        this.constitution = constitution;
        this.agility = agility;
        this.intelligence = intelligence;
        this.appearance = appearance;
        this.luck = luck;
    }

    public int getMaxHp(){
        return maxHp;
    }
    public void setMaxHp(int maxHp) { this.maxHp = maxHp; }

    public int getCurrentHp(){
        return currentHp;
    }
    public void setCurrentHp(int currentHp) { this.currentHp = currentHp; }

    public int getMaxMp(){
        return maxMp;
    }
    public void setMaxMp(int maxMp) { this.maxMp = maxMp; }

    public int getTrainedMaxMp() { return trainedMaxMp; }
    public void setTrainedMaxMp(int trainedMaxMp) { this.trainedMaxMp = trainedMaxMp; }

    public int getCurrentMp(){
        return currentMp;
    }
    public void setCurrentMp(int currentMp) { this.currentMp = currentMp; }

    public int getAttack(){
        return attack;
    }
    public void setAttack(int attack) { this.attack = attack; }

    public int getDefense(){
        return defense;
    }
    public void setDefense(int defense) { this.defense = defense; }

    public int getHitRate() { return hitRate; }
    public void setHitRate(int hitRate) { this.hitRate = hitRate; }

    public int getDodge() { return dodge; }
    public void setDodge(int dodge) { this.dodge = dodge; }

    public int getParry() { return parry; }
    public void setParry(int parry) { this.parry = parry; }

    public int getAttackSpeed() { return attackSpeed; }
    public void setAttackSpeed(int attackSpeed) { this.attackSpeed = attackSpeed; }

    public int getCritRate() { return critRate; }
    public void setCritRate(int critRate) { this.critRate = critRate; }

    public long getExperience() { return experience; }
    public void setExperience(long experience) { this.experience = experience; }

    public long getPotential() { return potential; }
    public void setPotential(long potential) { this.potential = potential; }

    public int getStrength() { return strength; }
    public void setStrength(int strength) { this.strength = strength; }

    public int getConstitution() { return constitution; }
    public void setConstitution(int constitution) { this.constitution = constitution; }

    public int getAgility() { return agility; }
    public void setAgility(int agility) { this.agility = agility; }

    public int getIntelligence() { return intelligence; }
    public void setIntelligence(int intelligence) { this.intelligence = intelligence; }

    public int getAppearance() { return appearance; }
    public void setAppearance(int appearance) { this.appearance = appearance; }

    public int getLuck() { return luck; }
    public void setLuck(int luck) { this.luck = luck; }

    public int getPostStrength() { return postStrength; }
    public void setPostStrength(int postStrength) { this.postStrength = postStrength; }

    public int getPostConstitution() { return postConstitution; }
    public void setPostConstitution(int postConstitution) { this.postConstitution = postConstitution; }

    public int getPostAgility() { return postAgility; }
    public void setPostAgility(int postAgility) { this.postAgility = postAgility; }

    public int getPostIntelligence() { return postIntelligence; }
    public void setPostIntelligence(int postIntelligence) { this.postIntelligence = postIntelligence; }

    public int getFlatAttackBonus() { return flatAttackBonus; }
    public void setFlatAttackBonus(int flatAttackBonus) { this.flatAttackBonus = flatAttackBonus; }

    public double getPctAttackBonus() { return pctAttackBonus; }
    public void setPctAttackBonus(double pctAttackBonus) { this.pctAttackBonus = pctAttackBonus; }

    public int getFlatHitBonus() { return flatHitBonus; }
    public void setFlatHitBonus(int flatHitBonus) { this.flatHitBonus = flatHitBonus; }

    public double getPctHitBonus() { return pctHitBonus; }
    public void setPctHitBonus(double pctHitBonus) { this.pctHitBonus = pctHitBonus; }

    public int getFlatParryBonus() { return flatParryBonus; }
    public void setFlatParryBonus(int flatParryBonus) { this.flatParryBonus = flatParryBonus; }

    public double getPctParryBonus() { return pctParryBonus; }
    public void setPctParryBonus(double pctParryBonus) { this.pctParryBonus = pctParryBonus; }

    public int getFlatDefenseBonus() { return flatDefenseBonus; }
    public void setFlatDefenseBonus(int flatDefenseBonus) { this.flatDefenseBonus = flatDefenseBonus; }

    public double getPctDefenseBonus() { return pctDefenseBonus; }
    public void setPctDefenseBonus(double pctDefenseBonus) { this.pctDefenseBonus = pctDefenseBonus; }

    public int getFlatDodgeBonus() { return flatDodgeBonus; }
    public void setFlatDodgeBonus(int flatDodgeBonus) { this.flatDodgeBonus = flatDodgeBonus; }

    public double getPctDodgeBonus() { return pctDodgeBonus; }
    public void setPctDodgeBonus(double pctDodgeBonus) { this.pctDodgeBonus = pctDodgeBonus; }

    public int getFlatMeditationBonus() { return flatMeditationBonus; }
    public void setFlatMeditationBonus(int flatMeditationBonus) { this.flatMeditationBonus = flatMeditationBonus; }

    public double getPctMeditationBonus() { return pctMeditationBonus; }
    public void setPctMeditationBonus(double pctMeditationBonus) { this.pctMeditationBonus = pctMeditationBonus; }

    public int getFlatLearningBonus() { return flatLearningBonus; }
    public void setFlatLearningBonus(int flatLearningBonus) { this.flatLearningBonus = flatLearningBonus; }

    public double getPctLearningBonus() { return pctLearningBonus; }
    public void setPctLearningBonus(double pctLearningBonus) { this.pctLearningBonus = pctLearningBonus; }

    public int getSkillPostStrength() { return skillPostStrength; }
    public void setSkillPostStrength(int skillPostStrength) { this.skillPostStrength = skillPostStrength; }

    public int getSkillPostAgility() { return skillPostAgility; }
    public void setSkillPostAgility(int skillPostAgility) { this.skillPostAgility = skillPostAgility; }

    public int getSkillPostParry() { return skillPostParry; }
    public void setSkillPostParry(int skillPostParry) { this.skillPostParry = skillPostParry; }

    public int getSkillMaxMpBonus() { return skillMaxMpBonus; }
    public void setSkillMaxMpBonus(int skillMaxMpBonus) { this.skillMaxMpBonus = skillMaxMpBonus; }

    public double getSkillMpToHpRatio() { return skillMpToHpRatio; }
    public void setSkillMpToHpRatio(double skillMpToHpRatio) { this.skillMpToHpRatio = skillMpToHpRatio; }

    // Total Post-natal Attributes (Base + Skill Bonus)
    public int getTotalPostStrength() { return postStrength + skillPostStrength; }
    public void setTotalPostStrength(int val) { /* Derived, do nothing */ }

    public int getTotalPostAgility() { return postAgility + skillPostAgility; }
    public void setTotalPostAgility(int val) { /* Derived, do nothing */ }

    public int getTotalPostConstitution() { return postConstitution; } // No skill bonus for Con yet?
    public void setTotalPostConstitution(int val) { /* Derived, do nothing */ }

    public int getTotalPostIntelligence() { return postIntelligence; } // No skill bonus for Int yet?
    public void setTotalPostIntelligence(int val) { /* Derived, do nothing */ }

    public int getTotalPostParry() { return skillPostParry; } // Parry is special, treated as bonus
    public void setTotalPostParry(int val) { /* Derived, do nothing */ }

    public void recalculateDerivedAttributes() {
        // Internal Force Limit
        int baseMp = this.constitution * 10; 
        this.maxMp = baseMp + this.skillMaxMpBonus;

        // Max HP
        // 气血=先天根骨*10 + 实时最大内力 * 转换比例
        int baseHp = this.constitution * 10;
        this.maxHp = baseHp + (int)(this.trainedMaxMp * this.skillMpToHpRatio);

        // Attack
        // 攻击=（先天臂力*0.3*（1+后天臂力）+数值攻击加成）*百分比攻击加成
        this.attack = (int)((this.strength * 0.3 * (1 + getTotalPostStrength()) + this.flatAttackBonus) * this.pctAttackBonus);

        // Hit Rate
        // 命中=（先天身法*0.5+数值命中加成）*百分比命中加成
        this.hitRate = (int)((this.agility * 0.5 + this.flatHitBonus) * this.pctHitBonus);

        // Parry
        // 招架=（先天臂力*0.1*（1+后天臂力）+数值招架加成）*百分比招架加成
        this.parry = (int)((this.strength * 0.1 * (1 + getTotalPostStrength()) + this.flatParryBonus + this.skillPostParry) * this.pctParryBonus);

        // Defense
        // 防御=（先天根骨*0.2*（1+后天根骨）+数值防御加成）*百分比防御加成
        this.defense = (int)((this.constitution * 0.2 * (1 + getTotalPostConstitution()) + this.flatDefenseBonus) * this.pctDefenseBonus);

        // Dodge
        // 躲闪=（先天身法*0.1*（1+后天身法）+数值躲闪加成）*百分比躲闪加成
        this.dodge = (int)((this.agility * 0.1 * (1 + getTotalPostAgility()) + this.flatDodgeBonus) * this.pctDodgeBonus);
    }
}
