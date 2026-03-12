package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.SkillManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Skill;
import com.mudgame.model.Player;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GetSkillsHandler extends GameCommandHandler {
    private static final String subtype = "get_skills";
    private final PlayerManager playerManager;
    private final SkillManager skillManager;

    public GetSkillsHandler(PlayerManager playerManager, SkillManager skillManager) {
        this.playerManager = playerManager;
        this.skillManager = skillManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info("engine: 接收到get_skills指令，正在获取玩家技能信息");
        Player player;
        List<Map<String, Object>> skillsList = new ArrayList<>();
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        try {
            player = playerManager.getPlayer(message.getPlayerId());
        } catch (Exception e) {
            Logger.error("获取玩家失败: " + e.getMessage());
            logs.add("ERROR: Failed to get player info.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), null, logs);
        }

        Map<String, Long> playerSkills = player.getSkills();
        if (playerSkills != null) {
            for (String skillId : playerSkills.keySet()) {
                try {
                    Skill skill = skillManager.getSkill(skillId);
                    if (skill != null) {
                        Map<String, Object> skillInfo = new HashMap<>();
                        skillInfo.put("id", skill.getId());
                        skillInfo.put("name", skill.getName());
                        skillInfo.put("description", skill.getDescription());
                        skillInfo.put("rarity", skill.getRarity());
                        skillInfo.put("type", skill.getType());
                        skillInfo.put("level", player.getSkillLevel(skillId));
                        skillInfo.put("percentage", String.format("%.2f", player.getSkillPercentage(skillId)));
                        skillsList.add(skillInfo);
                    }
                } catch (Exception e) {
                    Logger.warn("获取技能失败: " + skillId + ", " + e.getMessage());
                }
            }
        }

        results.put("player", player);
        results.put("skills", skillsList);

        Logger.info("engine: 成功获取玩家技能信息");
        logs.add("INFO: successfully get " + player.getId() + "‘s skills");

        return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);
    }

    @Override
    public String getType() {
        return type;
    }

    @Override
    public String getSubType() {
        return subtype;
    }
}
