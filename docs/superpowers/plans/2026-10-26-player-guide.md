# 玩家内容手册 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 让维护者和玩家知道当前能玩什么、故事如何触发、如何走完与获得奖励，并分清已接通和只有配置的内容。

**Architecture:** 手写玩家入口与推荐路线；生成器从当前 JSON 资源提取七卷全部副本、八城、任务、对话、武学和收藏。关键行为以已核对的 Go/前端代码为准，给出缺失触发、未绑定动作、同名副本等状态说明，保留完整剧透文本。

**Tech Stack:** Markdown、Python 标准库、Go 现有资源与运行时规则。

## Global Constraints

- 不修改玩法、奖励、存档或线上服务；不把规划和未接通配置写成可玩任务。
- 基准为当前 `resources/`；78 个副本、41 个任务、8 个世界、83 个技能、178 个物品、33 个称号均须覆盖。
- 全局 order 控制解锁，卷内 chapter 仅作目录标签；没有上一 order 时实际不锁定，不能照抄旧文档。
- 保留所有当前对话、机关结果、选择和结局；无剧情的后期模板如实标注，不杜撰原著剧情。

### Task 1: 核对实际入口

**Files:** `internal/manager/runtime_manager_dungeon.go`、`runtime_manager_dialogue.go`、`runtime_manager_player.go`、前端面板与全部资源（只读）。

- [x] 核对进入副本、结算、任务接交、逐句交谈、采集、技能和按钮规则。
- [x] 识别 4 条接通任务、37 条未接通任务、order 27 空缺、未绑定的后期听取任务动作，以及描述与实现不同的鬼市/北斗阵。

### Task 2: 编写玩家手册与全量故事攻略

**Files:** 新建 `docs/player/README.md`、`stories/*.md`、`worlds.md`、`quests.md`、`collections.md`、`content_status.md`；新建 `scripts/generate_player_guide.py`。

**Interfaces:** `python scripts/generate_player_guide.py` 更新事实页；`--check` 验证生成内容未漂移及文档内部链接/章节覆盖。手写入口不被生成器覆盖。

- [x] 入口写清按钮、起步路线、前九关体验、主题路线与卡关处理。
- [x] 每副本列出真实解锁条件、房间路线和门槛、人物、操作前置与分支、全目标、现场掉落、结算奖励、剧情原文。
- [x] 每世界列出出生点到各房间的路线、NPC 与交谈原文、学习/商店/采集点；41 任务均列出接交状态和目标旗标来源。
- [x] 列出全部技能的前置/残页实际来源/老师、装备和消耗品来源、33 称号；将未接通和薄剧情内容单列，避免误导。

### Task 3: 交叉校验与提交

**Files:** 更新 `docs/README.md`、旧 `docs/content/dungeon_flows.md`、`docs/DEVELOPMENT.md`。

- [x] 运行 `python scripts/generate_player_guide.py --check`，验证覆盖、链接与生成一致性。
- [x] 抽查新手关、复杂机关、分支副本、同 order、缺号 order、未接任务、同名副本与薄剧情后期关。
- [x] 保留旧流程中的九关速览但标记覆盖范围，修正错误总数和解锁规则，链接新手册。
- [x] 检查 diff，只提交本次文档与生成工具；中文本地提交，不推送。

## 验证记录

- 生成 12 个事实页，另有手写玩家入口；仓库 README、文档索引、前后端 README 和开发指导均已链接。
- `python scripts/generate_player_guide.py --check` 通过：内容无漂移，全部玩家页与内部锚点有效。
- 独立覆盖检查通过：78 副本章节、41 任务、65 公共房间、89 对话档案、83 技能、178 物品、33 称号。
- 89 份普通对话的所有开场和响应原文、69 行副本专用剧情台词均已核对收录；各副本机关、分支、重复和失败结果在卷内保留。
- 抽查惊雁宫星图、天刀双解、竟陵独占分支、鬼市三路、北斗七步、丝路护送和太虚幻境；明确未接动作、未接任务、序号空缺、同名副本与后期简短剧情。
- 修正旧九关速览的当前奖励与解锁说明，特别是长生诀残页同时存在现场与结算来源。
- 本次仅增加文档与生成工具，没有修改游戏资源、运行时代码、玩家存档或线上服务；未进行逐关实机通关。
