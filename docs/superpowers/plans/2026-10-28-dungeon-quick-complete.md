# 快速完成与内容回归修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 修复已审查回归，并在100%首通后提供同掉率、含途中固定收益的快速完成。

**Architecture:** manager在同一锁内复用结算函数，角色序号阻止重放。资源显式声明quickRewards，UI仅发送一次意图与服务端序号。原武学来源按已确认V2恢复，兼容修复在加载旧档时幂等执行。

**Tech Stack:** Go、Vue3、JSON资源、Python隔离协议冒烟。

## Global Constraints

- 权威规格：docs/design/dungeon_quick_complete_2026_10_28.md；修改顺序文档→测试→代码→验收回写。
- 本版免费，无次数或冷却成本；普通结算掉率15%，流程固定奖励必得。
- 不接触正式存档、不推送、不部署。composables不相互导入。
- 用户已要求实施，在本会话直接执行；不另行选择执行方式。

### Task 1: 恢复传承、保护与宫变目标

Files: resources/skills.json、npcs.json、items.json、dungeons/datang_gongbian.json；internal/manager/runtime_manager_player.go；internal/gamedata/martial_validation_test.go；新增manager回归测试。

- [x] 增加回归：无行宫旗标且持书的长生参悟失败；合法旧角色可装备；寇仲Kill拒绝；宫变读册前进度小于100。
- [x] 按0e85c40恢复品阶/来源/教师表，保留后续其他NPC和数值；实例Kill检查protected；宫变增加流程门禁与读册目标、改写串场文案。
- [x] 执行`go test ./internal/manager ./internal/gamedata ./internal/engine/runtime`，预期全部通过。

### Task 2: 共用掉落、领取记录兼容与确定性验证

Files: internal/manager/runtime_manager_dungeon.go、新建runtime_manager_dungeon_rewards.go、runtime_manager_persistence.go；奖励测试；删除fix_dangling.py。

- [x] 用`applyDungeonSettlementRewardWithRoll(p, settlement, func() float64)`验证0.149999掉落、0.15失败、失败不写领取、复刷古铜累加、空物品表保底。
- [x] 将发奖与概率分离，只有成功才标记FirstClaim；加载时修复五件历史兵器旗标并持久化一次性版本标记；完整秘本不使用一次性领取。
- [x] 以结算rewardItems与背包一致替换无效非负断言，运行manager测试。

### Task 3: 服务端快速完成与协议

Files: runtime_manager_dungeon_quick.go、model/types.go、storage/store.go、manager/persistence.go；engine/workflow.go；app/command_registry.go、handlers_dungeon_quick.go；engine/runtime/dungeon_manager.go、resources/dungeons/*.json。

Interfaces: `QuickCompleteDungeon(playerID, dungeonID string, expectedSequence int) (map[string]interface{}, error)`；目录`cleared/canQuickComplete/quickCompleteHint/quickCompleteSequence`；命令`quick_complete_dungeon`。

- [x] 测试首通资格、公共空闲限制、无回血/移动、流程奖励、15%共用规则、同序号只发一次、失败不扣序号、重载后重放拒绝。
- [x] 模板新增quickRewards，逐条登记目前无互斥和无成本的途中奖励，校验物品引用和正数；快速结算复用100%奖励并独立累加固定流程物品。
- [x] 增加角色持久化序号并补齐克隆/读写，注册命令并以选中角色而非请求playerId鉴权，返回实际奖励名称与数量。
- [x] 全量Go build/vet/test。

### Task 4: 按钮、真实消息与文档

Files: useDungeonView.js、GameInfoPanel.vue、GameView.vue、messageText.js、前端测试；scripts/tests/smoke_dungeon_quick.py；generate_player_guide.py及生成文档。

- [x] 前端测试一次点击只发送一次、未首通和忙碌拒绝、响应后刷新副本/属性/背包；按钮由服务端资格控制，展示当前免费和掉率。
- [x] WebSocket临时服务测试未通拒绝、离线仅为合成角色写通关、快速完成/重放/重新获取目录/重启序号一致；不写正式数据。
- [x] 运行前端test/lint/build，生成玩家手册再check；浏览器用合成账号验证按钮、收益展示和连续点击。
- [x] 回写本页和设计验收结果，清理临时进程，中文提交本地变更。

实际验收和已知经济边界见[实施记录](../../design/dungeon_quick_complete_2026_10_28.md#实施与验收结果)。本地提交前全部门禁通过，未推送部署。
