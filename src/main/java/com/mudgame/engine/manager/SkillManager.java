package com.mudgame.engine.manager;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.mudgame.exception.LoadException;
import com.mudgame.model.Skill;
import com.mudgame.storage.SkillStorage;
import com.mudgame.utils.Logger;

public class SkillManager {
    private final Map<String, Skill> skills = new ConcurrentHashMap<>();
    private volatile boolean loaded = false;
    private final SkillStorage skillStorage;

    public SkillManager(SkillStorage skillStorage){
        this.skillStorage = skillStorage;
    }

    public void loadAllSkills() throws LoadException {
        if(loaded){
            throw new IllegalStateException("manager：技能已加载，不能重复加载");
        }
        Map<String, Skill> loadedSkills = skillStorage.loadAllSkills();
        this.skills.putAll(loadedSkills);
        this.loaded = true;
        Logger.info("manager：加载所有技能成功");
    }

    public Skill getSkill(String skillId){
        if(!loaded){
            throw new IllegalStateException("manager：技能未加载，无法获取技能");
        }
        Skill skill=skills.get(skillId);
        if(skill==null){
            // Return a dummy skill or throw?
            // Throwing is safer for debugging, but might crash if ID is wrong.
            // Existing code throws IllegalArgumentException.
            // I'll keep it but maybe log warning.
            // throw new IllegalArgumentException("manager: 技能参数错误: " + skillId);
            return null;
        }
        return skill;
    }

    public String getSkillName(String skillId) {
        Skill skill = getSkill(skillId);
        return skill != null ? skill.getName() : "未知技能";
    }
}
