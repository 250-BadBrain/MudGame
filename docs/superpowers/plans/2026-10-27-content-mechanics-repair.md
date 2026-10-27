# 游戏内容与机制修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 修复现有内容无法触发、对话交易不一致、死亡恢复漏洞和物品操作缺口，让已配置玩法可正常完成。

**Architecture:** 保留现有manager/资源/前端注入结构；对话统一前置校验与历史，任务数据保留兼容别名；状态变更在同一锁及克隆事务内完成。所有运行验证使用临时目录与独立端口。

**Tech Stack:** Go、Vue3、JSON、Python标准库、PowerShell7。

## Global Constraints

- 先修改[设计决定](../../design/content_mechanics_review_2026_10_27.md)、formulas、economy和副本总纲，再改代码。
- 用户要求直接实施，本会话顺序执行，不要求额外选择执行方式；未安装的superpowers执行技能不阻塞。
- 38门武学V2、经验潜能曲线、商店价格和已持有物保留；不写正式存档、不部署、不push。
- composables互不import，由GameView注入；新增样式写所属组件。

## Task 1：状态、恢复与撤离

文件：`internal/manager/runtime_manager_player.go`、`runtime_manager_dungeon.go`，新增`runtime_manager_escape.go`；engine/app use_item结果与前端房间刷新。

- [x] 写行为回归：DEAD→Stop/Leave仍DEAD；COMBAT→Leave拒绝；Meditate→Stop循环MP不变；首次10秒正常恢复；永久已修大于当前上限时复活MP≤有效上限。
- [x] 开始恢复只设置状态与10秒倒计时；Heal要求空闲、有效MP≥8；复活MP=`eff.TrainedMaxMP/2`，HP=`eff.MaxHP/2`。
- [x] 将退出结算抽为`leaveDungeonNoLock(players map[string]*model.Player,p *model.Player)`，公开LeaveDungeon先拒绝DEAD/COMBAT；烟尘在COMBAT内显式调用内部退出，扣1包与结算同次保存。
- [x] UseItem结果增加实际房间变化刷新，确保旧房间离开、新房间进入、房间信息和任务/背包刷新正确；无烟尘或非战斗不消耗。
- [x] `go test ./internal/manager ./internal/engine`，包含烟尘未满通关不获绝学/旗标及PvP目标清理。

## Task 2：对话与交易

文件：`model/types.go`（DialogueOption.Repeatable）、`runtime_manager_dialogue.go`，配方资源；新增对话回归。

- [x] 回归ChooseDialogue不能重复一次性选项；失败TakeItems、失败任务交付不扣钱/发物/写历史；not_started条件不能忽略Flag/Items。
- [x] `dialogueOptionAvailableNoLock`统一历史、条件、takeItems、takeMoney、任务接交资格；先验证再提交。Talk/Choose共用一次性记录，repeatable不被历史阻断。
- [x] 自动Talk跳过消耗服务，返回可明确点击的服务choices；配方按设计文档改成本与铜钱文字，repeatable只赋给有成本的服务。
- [x] 游戏日志增加`dialogue_choice`选项，经现有choose_dialogue发送npcId/optionId；选后刷新服务、物品和属性，重复服务再次点击允许。
- [x] `go test ./internal/manager -run 'Dialogue|Herbal|Recipe'`；协议测试重复配方和一次性任务奖励。

## Task 3：完整任务闭环与城市内容

文件：`quests.json`、`dialogues.json`、`npcs.json`、worlds；新增`runtime_manager_quests.go`承接任务规则，gamedata资源校验。

- [x] 41定义中3项aliasOf：xinggong_after→xinggong_aftermath、kongming_dao→kongming_daozhen、longgong_yi→longgong_yiji；旧完成别名同规范已完成。
- [x] 10条试炼目标依次修正为datang_xb2/fuyu_ht2/posui_sh2/bianhuang_dz2/luoshui_tg2/yangzhou_ym2/datang_tx2/fuyu_yy2/posui_th2/bianhuang_bf2的满通关旗标。
- [x] 粮道悬赏委托人改office_clerk；图录/空明辨读改ca2_monk。补ly_refugee_camp哑婆NPC及对应对话，线索原旗标保留；漕银补两位证人目标并保留苇荡实物线索。
- [x] 每个规范任务在指定NPC对话加入`requires.questStatus=not_started`的start和active且目标满足的complete；GetQuests返回38项含可接任务、接交位置与实际进度，别名不单列。
- [x] model.Room/rawRoom读取Rumors，listen_rumor优先当前房间配置；复用旧通用兜底。
- [x] 全部规范任务表驱动测试接交、已完成不可重复、错误地点/目标不满足拒绝；加载检查规范接交和目标来源、别名无环、Npc可达。

## Task 4：副本动作与叙述

文件：35个缺简报的dungeons/*.json、鬼市/北斗文案、engine/runtime与资源校验测试。

- [x] 将未绑定listen_brief加入动作声明的rooms.actions；选择分支不误加为公共按钮。
- [x] 鬼市说明为三路皆可探索；北斗说明越序不生效，统一天枢/天璇/天玑/天权/玉衡/开阳/摇光名称；保持稳定ID。
- [x] 校验每个普通主动作可从房间或可达NPC对话触发，未绑定动作使资源测试失败；遍历78关验证真实DoRoomAction简报可以推进状态。

## Task 5：强化与物品界面

文件：`runtime_manager_enhance.go`、GetBackpack/GetEquipment、GameInfoPanel/useBackpackActions/GameView。

- [x] 回归未持有物、COMBAT、DEAD、潜能以外钱材不足不发生强化；背包或合法装备拥有均可，最高5。
- [x] 后端返回当前等级、下级钱材成本和增益；背包与装备均有强化确认，显示资源和结果，不让玩家盲扣材料；成功刷新背包/装备/属性。
- [x] 金钱文案统一铜或现有formatMoney，不改底层金额；烟尘使用说明清楚战斗撤离与部分结算。

## Task 6：交付

- [x] `go test ./...`、`go vet ./...`、`go build ./...`；前端lint/build及有意义的状态/意图回归。
- [x] 独立端口真实WebSocket冒烟14项通过。
- [ ] 网页确认任务/配方/强化/烟尘并保存截图：Computer Use因无法可靠识别当前浏览器网址终止，本轮未通过界面验收。
- [x] 攻略生成器同步别名统计、可接任务、repeatable服务、传闻、撤离及新增入口；生成12页并`--check`，手动同步新手路线。
- [x] 回写实际覆盖、限制与部署状态，停止测试进程，中文本地提交，不push。

验收证据和限制见[复核报告](../../design/content_mechanics_review_2026_10_27.md)。实际数据为38个唯一任务及3个别名、35段补齐简报；Go全量检查、6项前端回归、21项Python回归和14项协议检查通过。临时服务已停，正式服务未改。
