# 武功体系 V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 实施已授权的38门首版武学、统一修炼与五位置生效、配套装备、战斗/经济修订及可回退的离线迁移。

**Architecture:** 静态资源声明位置、兵器类别和传承来源，后端集中校验初学、修炼、装备与属性。新角色直接使用V2；旧存档由独立只读预览/停服应用工具迁移，服务拒绝未经迁移的旧档。前端展示全目录、资格和生效原因，直接发自主修炼/卸下指令。

**Tech Stack:** Go 1.26、Vue 3/Pinia/Vite、Python标准库、JSON资源、PowerShell 7。

## Global Constraints

- 用户已授权按修订方案实施；38门不是永久容量限制。
- 同一武功不拆多卷，不增加升星、合卷或突破材料。
- 保留经验/逐级潜能曲线、5000铜初始资金（50两）及既有物品所有权。
- 所有运行测试使用临时存档/端口；不部署、不写正式存档、不push。
- 本会话直接执行，不另开用户任务；未安装的superpowers执行技能不作为阻塞，按本计划逐项验证。

## 文件与接口

| 职责 | 文件 |
|---|---|
| 字段与落盘 | `MudGame-server/internal/model/types.go`、`internal/storage/store.go`、`internal/manager/runtime_manager_persistence.go` |
| 统一规则 | 新建`internal/manager/runtime_manager_martial.go`，改`runtime_manager_player.go`与`runtime_manager_effective.go` |
| 指令 | `internal/app/command_registry.go`、`handlers_phase2.go`、`internal/engine/workflow.go` |
| 资源校验 | `internal/gamedata/static_loader.go`及测试 |
| 战斗与投放 | `runtime_manager_combat.go`、`runtime_manager_dungeon.go`、`internal/engine/runtime/dungeon_manager.go` |
| 迁移 | `resources/migrations/martial_v1.json`、`resources/migrations/migrate_martial_v2.py`，部署脚本及回归测试 |
| 前端 | `GameInfoPanel.vue`、`useShopLearn.js`、`useBackpackActions.js`、`GameView.vue`、`messageText.js` |
| 文档与冒烟 | formulas/curriculum/learning/economy、玩家攻略生成器、`scripts/tests/` |

核心接口约定：

```go
func skillRequirementError(p *model.Player, def model.SkillDef) error
func skillTrainingCap(p *model.Player, def model.SkillDef) int
func skillActiveReason(p *model.Player, def model.SkillDef) string
func (r *RuntimeManager) PracticeSkill(playerID, skillID string) error
func (r *RuntimeManager) UnequipSkill(playerID, slot string) (string, error)
```

资格函数只检查根基、读书、传承，不修改角色。生效函数另外检查已学、位置、兵器。属性计算不修改持久化数据；实际换装才夹紧当前资源。

## Task 1: 权威文档与资源

- [x] 将批准的数值和规则写入权威文档；提案同步实施状态。
- [x] 保存83技能/69秘籍的旧版迁移快照，含原类型、价值、映射，不读取真实存档。
- [x] 改38门技能、18种章节完整秘本及7种迁移兼容秘本、9件新武器、老师、章节投放与文案；活动资源中的旧分卷引用归零。
- [x] 静态校验位置、武器类别、基本前置、老师、秘本和来源副本闭环，不限定总数。

验证：`go test ./internal/gamedata ./internal/engine/runtime`。保留的旧内容断言按新契约更新，不能删除资源一致性门禁。

## Task 2: 后端修炼与生效

- [x] 持久化MartialVersion、ActiveArts、恢复品冷却；克隆时深拷贝，创建新角色标记V2。
- [x] 实现上述统一接口；老师免费入门，已学技能在安全公共房间自主修炼。
- [x] 10秒预算逐级消费、越级结算不吞潜能；断线暂停，达上限立即停止。
- [x] 五位置过滤、暗器单主武器、动态属性重算、先夹有效内力再转血。
- [x] 拆命中/招架概率、保留小数暴击、打坐成长、共享用药冷却、普通采集修正。
- [x] 最近前序副本解锁；传承/新兵器按结算和领取记录防重复。

验证数据：潜能10、拳脚1、经验足够，一次结算到11且余额0；基本内功120、读书120、长生100、已修2020时MaxMP2020/MaxHP1033；招架56/78对命中140分别28.57%/35.78%。完整测试覆盖失败不扣书、资格不符不生效、战斗拒绝换装、换功不回血/丢永久内力、药与干粮共冷却。

## Task 3: 离线迁移与部署事务

- [x] 迁移工具默认仅输出报告；显式停服应用先验证全员，再备份、暂存、逐文件原子替换；失败恢复已改文件。
- [x] 基本功先迁、同门折算投入再合并、退役补偿、上限溢出退潜能；书本/装备迁移不丢物，未知或异常数据阻止应用。
- [x] 已迁角色幂等跳过；未迁旧角色由服务拒绝加载，避免静默返款或新程序自动覆盖。
- [x] 部署显式迁移选项；停服备份后迁移，失败连同迁移前数据回滚，保留逐角色报告。

验证：临时目录合成单门/多卷/未达绝学条件/重复秘本/双武器/异常进度，预览原文件字节不变；应用潜能守恒；二次应用字节不变；失败恢复快照。运行`python -m unittest discover -s scripts/tests -v`。

## Task 4: 前端与玩家说明

- [x] 技能页显示全目录、固定品阶、等级、来源、条件、生效原因；已学可修炼，命名武学可装备/卸下。
- [x] 新增practice/unequip_skill，原learn已学请求兼容；成功后静默刷新技能、属性和背包。
- [x] 暗器并入主武器显示；恢复品显示服务端冷却，重连校正；五档颜色仅武学/秘本使用。
- [x] 更新玩家攻略生成器的前序、奖励、修炼和加成解释并重新生成，人工同步新手路线。

验证：前端lint/build；真实WebSocket走创角→拜师→修炼→传承失败/成功→换装→用药→重连。浏览器验证技能操作和冷却，使用临时测试账号与端口。

## Task 5: 总验收与收尾

- [x] `go build ./...`、`go vet ./...`、`go test ./...`。
- [x] `npm run lint`、`npm run build`；`python scripts/generate_player_guide.py --check`。
- [x] 分阶段九流派代表关卡战斗、药耗和传承前通关检查；失败时调已有数值，不造新分卷。
- [x] 停止临时测试进程，隔离验收档留在Git忽略的artifacts内，更新实施记录与部署说明，中文本地提交。

本计划记录完成证据后勾选；不把公式复算替代实际消息流、存档往返和通关验证。

## 完成证据

2026-10-27本地实现并验收，未部署、未写正式存档、未push。[实施记录](../../design/martial_arts_v2_implementation.md)包含逐项测试、162场九流派首领战、真实协议13项流程、浏览器截图及首发迁移方法。完整秘本的成功/失败参悟、战斗换装拒绝使用Go回归覆盖；协议冒烟验证自然时间修炼、冷却、重连和真实副本战斗。未声明78关连续实机通关或五人并发压测完成。
