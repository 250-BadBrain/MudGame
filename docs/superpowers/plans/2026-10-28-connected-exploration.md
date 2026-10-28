# Connected Exploration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 将三件城市旧事接入现场，六门已有武学提供替代解法，见闻说明实际下一步，同步玩家攻略。

**Architecture:** 沿用对话旗标与机关状态；PuzzleAction增requiredSkills，统一检查已学等级与原资格。Journey元数据挂在现有DialogueDef，GetJournal派生关联卡片，不新增实体与玩家字段。GameQuestPanel展示关联/下一步，其余历史兼容。

**Tech Stack:** Go、JSON、Vue、Python、Playwright。

## Global Constraints

- 规格见docs/design/connected_exploration_2026_10_28.md；原实体集合和奖励冻结，普通路线不需要任何新条件。
- 已授权本会话顺序实施，不派子代理，不推送部署，不使用正式存档。执行子技能不在本机技能目录时沿用本计划逐项执行和测试，不阻塞已授权工作。
- composables不互相import。直接修改本轮资源，不重跑会覆盖后续编辑的enrich_existing_content.py。

### Task 1: 条件和内容

**Files:** engine/runtime/dungeon_manager.go；manager/runtime_manager_content_evidence.go、runtime_manager_player.go、runtime_manager_room.go；gamedata/content_evidence_validation.go；resources/dialogues.json和7个受影响副本JSON；manager/runtime_manager_connected_exploration_test.go。

- [x] 先写六门武学实际DoRoomAction测试；动作前置按规格表，assert未学blocked、状态未变，学会且根基满足后success；测试资格缺失/错误房间、无资金背包技能变化。
- [x] `PuzzleAction.RequiredSkills map[string]int`及文件字段`json:"requiredSkills"`，loadPuzzleActions复制；`contentSkillsMet(p, required)`检查等级与`skillRequirementError`；executePuzzleAction和可见性过滤都调用。
- [x] 按规格写6个`connection_skill`动作，复用原观察/支撑/比对状态；写6个`connection_follow_<side>`明确对话与`connection_case_<side>`现场操作，两者均要求原分支旗标，后者还要求追问旗标。
- [x] `go test ./internal/manager -run 'Connected|ExpandedChapter|Evidence'`；预期新解法成功、旧78关流程仍通过。更新gamedata静态校验：已知技能/正数等级，关联引用真实NPC/旗标/副本动作。

### Task 2: 指引与玩家界面

**Files:** model/types.go（DialogueJourney/Branch）；manager/runtime_manager_content_evidence.go；components/GameQuestPanel.vue；manager/runtime_manager_connected_exploration_test.go。

- [x] 对话上挂Journey：Title/DungeonID/StartFlag/Witnesses/ItemID/Branches，分支含Flag/NPCID/OptionID/FollowFlag/ActionID/UsedFlag；GetJournal追加group=journey、progress、state、related、next字段，id用现有NPCID派生。
- [x] 测试新角色不显示、证人缺失只指缺失对象、物品缺失提示、单结论只提示所选追问、已用情报/自行通关均完成，不展示未选结果；只在当前实际条件具备时说可入现场。
- [x] GameQuestPanel用`journal.filter(entry => entry.group === 'journey')`展示进度卡；历史用其余项，next用正文样式换行，保留details和空提示，不增加跨composable依赖。

### Task 3: 文档和验收

**Files:** scripts/generate_player_guide.py；docs/player/connected_exploration.md、README.md与生成页；docs/SYSTEMS.md/DEVELOPMENT.md/README.md；scripts/tests/smoke_connected_exploration.py、browser_connected_exploration.cjs、browser_dungeon_quick.py。

- [x] 生成器用技能中文名输出requiredSkills、可选解法和Journey关联；玩家新页说明三条线六分支、六门武学的地点/门槛/作用、普通解法仍可用、见闻入口，入口文档链接。
- [x] `go build ./...`、`go vet ./...`、`go test ./...`；`npm test`、`npm run lint`、`npm run build`；`python -m unittest discover -s scripts/tests -v`；生成手册并`--check`。
- [x] 独立真实WS覆盖城市选择→追问→现场及武学动作和重启指引；网页点击关联卡、追问与武学按钮并检查截图/页面错误。回归原内容深度和审查行为。
- [x] 回写验收证据与边界、git diff --check，清理测试服务，中文本地提交，不推送部署。
