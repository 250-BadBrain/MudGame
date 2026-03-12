package com.mudgame.storage;

import java.io.IOException;
import java.io.InputStream;
import java.util.Collections;
import java.util.Map;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.exception.LoadException;
import com.mudgame.model.Skill;
import com.mudgame.utils.Logger;

public class SkillStorage {
    private static final String SKILLS_DIR="skills.json";

    private final ObjectMapper objectMapper;

    public SkillStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    public Map<String, Skill> loadAllSkills() throws LoadException {
        InputStream tempIs = SkillStorage.class.getClassLoader().getResourceAsStream(SKILLS_DIR);
        if (tempIs == null) {
            tempIs = SkillStorage.class.getResourceAsStream("/" + SKILLS_DIR);
        }

        if(tempIs == null) {
            Logger.error("storage：未找到技能配置文件: " + SKILLS_DIR);
            throw new LoadException("未找到技能配置文件: " + SKILLS_DIR);
        }

        try(InputStream is = tempIs) {
            TypeReference<Map<String, Skill>> typeRef = new TypeReference<Map<String, Skill>>() {};   
            Map<String, Skill> skills = objectMapper.readValue(is, typeRef);

            // Set the ID for each skill based on the map key
            for (Map.Entry<String, Skill> entry : skills.entrySet()) {
                Skill skill = entry.getValue();
                if (skill.getId() == null) {
                    skill.setId(entry.getKey());
                }
            }

            Logger.info("storage：成功加载技能配置，数量: " + skills.size());

            return Collections.unmodifiableMap(skills);
        } catch (IOException e) {
            throw new LoadException("加载技能失败: " + e.getMessage(), e);
        }
    }
    
}
