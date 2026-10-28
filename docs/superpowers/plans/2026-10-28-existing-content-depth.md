# 既有资源内容深化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 冻结实体ID，通过专属人物职责、器物解读、在地调查和章节交互丰富内容。

**Architecture:** 现有JSON承载具体内容，对话/机关扩展显式选择与证据条件；manager统一验证持有，任务面板只回看已取得的见闻。编写确定性内容编辑脚本和冻结清单，保留可审查的逐条映射。

**Tech Stack:** Go、JSON、Vue、Python、Playwright。

## Global Constraints

- 设计见docs/design/existing_content_depth_2026_10_28.md。不增加/删除物品、NPC、房间、世界、副本、任务、武学或势力ID；不改奖励和掉率。
- 无稀有掉落强制首通门槛，保留便利商店和技能来源；安全区/被动NPC/非致死切磋不变。
- 用户已授权本会话实施；顺序执行，不派子代理，不部署推送，不碰正式存档。

### Task 1: 条件与见闻

Files: model/types.go、gamedata/static_loader.go、engine/runtime/dungeon_manager.go、manager/dialogue.go、dialogue_rules.go、player.go、engine/workflow.go、app查询handler；GameQuestPanel.vue/GameView.vue。

- [x] 对话增加`Explicit bool`及Requires.Flags/PresentItems/Skills；自动Talk跳过explicit，选择仍执行统一权威校验。
- [x] `heldItemCount(p,id)`只计背包与合法槽位装备；机关requiredFlags和presentItems与房间/状态校验叠加。测试未持有、已穿戴、错误旗标、伪造分支均拒绝且无副作用。
- [x] `GetJournal(playerID) ([]map[string]string,error)`从已取得的调查/对话旗标提取响应；get_quests附带journal，任务面板展示。测试不会泄露未取得的选项。

### Task 2: 内容与覆盖

Files: scripts/enrich_existing_content.py；resources现有items/npcs/dialogues/worlds/dungeons；gamedata testdata冻结清单与内容测试；docs/content逐条报告。

- [x] 保存71085f3资源ID与奖励基线；编辑器运行前后验证集合相同，禁止添加新实体。
- [x] 写明公共人物专长、42未出场NPC的归位、135件物品独有出示/解读、65房间在地线索；既有教学与交易不移除。
- [x] 38个通用开门关用具体观察与决策深化，保留open_gate等既有主动作ID和目标；原38关加双龙会、护送商队共40关补专属可选复盘，不动主线权重。不同机关线索、选择、失败反馈逐条写明。
- [x] 自动遍历所有扩写解法、错误重试、主线不需稀有物品，核对无悬空NPC/物品与奖励改变。

### Task 3: 交付验收

Files: generator、玩家手册和NPC/物品用途报告、专项真实协议/浏览器测试。

- [x] 更新生成器显示NPC专长、出示途径、章节差异与见闻入口；完成后生成并check。
- [x] Go build/vet/test、前端test/lint/build、Python迁移/部署回归，真实协议和独立浏览器验证；截图检查、回写证据、中文本地提交。
