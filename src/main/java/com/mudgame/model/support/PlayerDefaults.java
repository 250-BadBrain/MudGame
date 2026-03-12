package com.mudgame.model.support;

import com.mudgame.model.Attributes;
import com.mudgame.model.Player;
import java.util.ArrayList;
import java.util.Random;

public final class PlayerDefaults {
    private static final Random random = new Random();

    private PlayerDefaults() {}

    /**
     * 用于初始化一个新创建的角色对象
     * @param characterId 新角色的唯一ID (作为 Player.id 和存档文件名)
     * @param characterName 新角色的名称 (作为 Player.characterName)
     * @param userId 角色所属的账户ID (作为 Player.userId)
     * @param strength 臂力
     * @param constitution 根骨
     * @param agility 身法
     * @param intelligence 悟性
     * @return 初始化的玩家对象
     */
    public static Player create(String characterId, String characterName, String userId, int strength, int constitution, int agility, int intelligence) {
        Player p = new Player();
        p.setId(characterId);          // 设置角色的唯一ID
        p.setCharacterName(characterName); // 设置角色的显示名称
        p.setUserId(userId);           // 设置角色所属的账户ID
        p.setRoomId("ly_00");          // 默认出生点
        
        // 随机生成容貌 (18-25)
        int appearance = generateAppearance();
        // 随机生成福禄 (20-25)
        int luck = generateLuck();

        // 初始化属性
        // 暂时给予默认的基础战斗属性，先天属性由创建时指定
        Attributes attrs = new Attributes(
            100, 100, // HP
            100, 100, // MP
            20,       // Attack (Initial 20)
            20,       // Defense (Initial 20)
            20,       // HitRate (Initial 20)
            20,       // Dodge (Initial 20)
            20,       // Parry (Initial 20)
            4,        // AttackSpeed (Initial 4)
            1,        // CritRate (Initial 1%)
            0,        // Experience
            0,        // Potential
            strength, constitution, agility, intelligence,
            appearance, luck
        );
        p.setAttributes(attrs);
        
        p.setMoney(10000); // Initial 1 Gold (10000 Copper)
        p.setItems(new ArrayList<>());
        
        // Initialize skills with Basic Internal and Basic Fist at level 10
        java.util.Map<String, Long> initialSkills = new java.util.HashMap<>();
        // Calculate potential for level 10
        // Total(L) ≈ 0.0076923 * L^2.6 + 200 * L
        // For L=10: 0.0076923 * 10^2.6 + 2000 ≈ 0.0076923 * 398 + 2000 ≈ 3 + 2000 = 2003
        // Let's use the Player method logic or just a safe value.
        // 2500 potential should be enough for level 10.
        long potForLvl10 = 2500L; 
        initialSkills.put("basic_internal", potForLvl10);
        initialSkills.put("basic_fist", potForLvl10);
        p.setSkills(initialSkills);

        // 先根据初始技能重算派生属性，再把初始气血内力校准到满值。
        p.updateSkillBonuses();
        attrs.setTrainedMaxMp(attrs.getMaxMp());
        attrs.recalculateDerivedAttributes();
        attrs.setCurrentMp(attrs.getTrainedMaxMp());
        attrs.setCurrentHp(attrs.getMaxHp());
        
        return p;
    }

    private static int generateAppearance() {
        // 18-25, approximate normal distribution
        // 18: 5%, 19: 10%, 20: 15%, 21: 20%, 22: 20%, 23: 15%, 24: 10%, 25: 5%
        int roll = random.nextInt(100);
        if (roll < 5) return 18;
        if (roll < 15) return 19;
        if (roll < 30) return 20;
        if (roll < 50) return 21;
        if (roll < 70) return 22;
        if (roll < 85) return 23;
        if (roll < 95) return 24;
        return 25;
    }

    private static int generateLuck() {
        // 20-25, arbitrary distribution
        // 20: 10%, 21: 20%, 22: 30%, 23: 20%, 24: 15%, 25: 5%
        int roll = random.nextInt(100);
        if (roll < 10) return 20;
        if (roll < 30) return 21;
        if (roll < 60) return 22;
        if (roll < 80) return 23;
        if (roll < 95) return 24;
        return 25;
    }
}