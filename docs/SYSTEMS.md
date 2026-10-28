# 游戏系统总览与开发指引

本文是**全部游戏系统的目录**：每个系统是什么、代码与数据在哪、如何扩展、开发时的注意事项。用于新会话/新 agent 接手时快速定位，配合 [DEVELOPMENT.md](DEVELOPMENT.md)（仓库结构与标准流程）使用。

面向实际游玩的内容、故事触发与剧情全文见 [玩家手册](player/README.md)；本页仍作为系统开发索引。

> **数值不在本文**：所有公式以 [formulas.md](design/formulas.md) 为唯一权威（含公式→代码对照表），资源产出与定价以 [economy_design.md](design/economy_design.md) 为唯一权威。本文只讲机制与开发路径。

## 系统总目录

| 系统 | 核心代码 | 数据文件 |
|---|---|---|
| 账号与角色 | `app/handlers_system_game.go` | storage: users/players |
| 秘籍装备 | `manager/runtime_manager_player.go`（EquipSkill） | skills.json requirements.prerequisite |
| 装备强化 | `manager/runtime_manager_enhance.go`（EnhanceItem） | Player.EnhancedItems（0-5级，费用递增，仅当前合法装备生效） |
| 称号系统 | `manager/runtime_manager_titles.go`（getPlayerTitles） | titles.json（33个旗标→称号映射） |
| 马车跨城 | `manager/runtime_manager_player.go`（EnterWorld 跨城100铜、同城免费） | worlds/*.json（8座城市） |
| 多世界 | `manager/runtime_manager_player.go`（EnterWorld） | worlds/*.json |
| 房间与世界 | `manager/runtime_manager_room.go` | `resources/worlds/*.json` |
| 副本 | `manager/runtime_manager_dungeon.go` + `engine/runtime/dungeon_manager.go` | `resources/dungeons/*.json` |
| 战斗 | `manager/runtime_manager_combat.go` | — |
| 属性 | `model/attributes.go` + `manager/runtime_manager_effective.go` | — |
| 秘籍装备 | `manager/runtime_manager_player.go`（EquipSkill）+ `runtime_manager_effective.go`（装备过滤） | skills.json requirements.prerequisite |
| 技能与学习 | `manager/runtime_manager_player.go`（ProcessStatusTick） | `resources/skills.json`（38门首版，可扩展） |
| 物品与背包 | `manager/runtime_manager_player.go`（UseItem/Buy/SellItem） | `resources/items.json` |
| 商店 | 同上 + `manager/runtime_manager_static_npc.go` | NPC 模板 `shopItems` 字段 |
| 状态与恢复 | `manager/runtime_manager_player.go`（ProcessStatusTick/Stop） | worlds 房间 `actions` |
| 对话 | `manager/runtime_manager_dialogue.go` | `dialogues.json` / `dungeon_dialogues.json` |
| 任务 | `manager`（flag 驱动） | `quests.json` |
| 势力声望 | 结算与 NPC 模板 | `factions.json` |
| 聊天 | `app` chat 通道 | — |
| 事件推送 | `app/server.go` 广播 | — |
| 持久化 | `storage/store.go` | `data/servers/server1/` |

---

## 1. 账号与角色

注册/登录/删号/创角/选角/重连。会话与玩家绑定由 app 层管理（`bindSessionUser`/`setSessionPlayerID`，重连清理旧绑定）。

- **开发**：改鉴权行为在 `handlers_system_game.go`；创角校验（属性总和 80、名字 2~16 字节）在 `manager/runtime_manager_system.go` CreateCharacter。
- **注意**：客户端传的 userId 不可信，handler 一律以会话绑定的用户名为准；登录必须清掉旧的 playerID 绑定（防会话串号）。

## 2. 房间与世界

静态地图：世界 → 房间。**多世界**：`worlds/*.json` 每世界一文件（luoyang/changan），`enter_world {worldId}` 跨城直达出生点（name/description/exits/npcs/actions/tags/safeZone/function）。移动 `Move(playerID, direction)`；副本外 NPC 挂在房间 `npcs` 数组。

- **加房间/改地图**：编辑 `worlds/luoyang.json`（或新世界文件），跑 `go test ./internal/gamedata/`——`validateRooms` 会拦重复房间 ID、悬空出口、出生点缺失。
- **注意**：世界是**公共空间不做战斗资源产出**（设计原则，见 economy_design.md §0）；采集/帮工通过房间 `actions: ["gather"/"work"]` 声明，收益在 ProcessStatusTick 结算（30s/60s 忙碌）。

## 3. 副本系统（内容开发主战场）

**模板**（`resources/dungeons/<id>.json`，每章一个文件）：order（章节序）/chainId/chainName/chapter、rooms、objectives、puzzleActions、puzzleInitialStates、rewards、可选 unlockConditions。

**实例**：进入时整体复制生成（房间 ID 加实例后缀，`InstanceRoomIDNoLock` 做模板→实例映射）；离开即销毁；重复进入全部重置。

### 3.1 目标与完成度

- 类型：`kill`（targetId + required 数量）/ `puzzle`（`targetId: "<puzzleId>:<state>"`，**必须带 required:1**）/ `enter_room`（到达房间）。
- 权重数学：**非独占权重之和 + 每个独占组内最大权重 = 100**。`optional: true` 不计入分母；同 `exclusiveGroup` 的目标只计其一（两线任一可 100%，见竟陵之围）。
- 结算：经验/潜能/金钱按完成度折算；物品与旗标仅 100% 发放；**100% 时自动写 `dungeon_cleared_<id>` 玩家旗标**。

### 3.2 章节链解锁

第一组开放；目标序号需最近存在的前一组任意副本100%结算（读 `dungeon_cleared_*`），缺号不自动开放。例如28需26；显式unlockConditions另行检查。额外条件用显式 `unlockConditions`（type: flag，AND 语义，如拦江需两面旗）。锁定态由 `dungeonLockStateNoLock` 计算，前端展示 🔒 + lockHint。

### 3.3 机关（puzzleActions）

状态机动作：`puzzleId` + `fromStates` + `nextState`，可选 `choices`（分支动作按 choice id 再定义，**分支动作不用登记进房间 actions**）、`setFlags`（**写入玩家旗标**——Boss 破招读的就是它）、`giveItems`、`requiredDefeatNpcs`、`repeatMessage`。主机关动作**必须列入所在房间的 `actions` 数组**，否则前端不可见、DoRoomAction 报 action_not_available。

### 3.4 剧情动作（story action）

带 `dialogueNpcId` 的机关动作，四件套缺一不可：① puzzleActions 里定义（专用琐碎 puzzleId，如 new→told）；② NPC 模板有 `dialogueId`；③ `dialogues.json` 有该对话条目；④ `dungeon_dialogues.json` 按副本 ID + 动作 ID 提供 `lines`（≥2 句）与 `doneMessage`。Talk 逐句推进（DialogueProgress）。

### 3.5 NPC 保护与 Boss

- 副本 NPC 一律 `instanceOnly: true`，ID 前缀按副本（csj_/nj_/jyg_/bh_/yg_/lj_/jl_/td_…）。
- 叙事角色 `combatMode: "protected"`（击杀/切磋直接拒绝）；氛围角色 `nonCombat: true`（"并无战意"）；**defeat 模式 Boss**（`combatMode: "defeat"`）：被打至 25% 血即认负，留在房内不死、不可再击杀（方夜羽、宋缺）。
- **破招**：Boss 配 `weaknessFlag` + `weaknessModifiers`；玩家持有该旗标（通常由机关 setFlags 写入）并对其 Kill 时应用削弱。

### 3.6 新增副本 checklist（按序）

1. 文档先行：总纲表格加行 + §三逐章设计 + economy 收益表；实现后生成玩家手册对应卷事实页并核对入口与剧情。旧 `dungeon_flows.md` 只保留前九关摘要。
2. npcs.json 加 NPC（前缀、instanceOnly）、factions/skills/items 按需。
3. 写 `dungeons/<id>.json`（参考：复杂机关 `datang_yanggong.json`、多步+选择 `posui_jingyan.json`、独占分支 `datang_jingling.json`、试炼型 `datang_tiandao.json`）。
4. **登记引擎测试三处硬编码**：`dungeon_manager_test.go` 的 expected 表（房间数/chain/chapter）、story 对话总数（covered 计数）、其余断言按需。
5. 全流程测试（模板见 `runtime_manager_jingling_test.go`）+ 冒烟。
- **注意**：`TestStaticResourcesAreConsistent` 会拦截一切悬空引用（NPC/物品/技能/对话/动作/目标）；杀权重 ≥50、解谜权重 ≤20（TestCombatFocusedDesignConstraints）。

## 4. 战斗系统

Kill（决斗，至死）/ Spar（切磋，双方须满血，单次伤害上限为受击方有效MaxHP/5，至少留1血，认输后取消反击）。战斗 tick 2 秒；状态 tick 1 秒。敌对NPC默认不主动交战；Aggressive仅标识敌对，当前没有自动开战例外。先手值决定开战首击。公式全部在 formulas.md §2（命中/招架/伤害/暴击）。

- **注意**：药品**可在战斗中使用**（核心消耗品，别加回 COMBAT 拒绝）；死亡 relive 后虚弱 300 秒（战斗衍生减半）回副本检查点/洛阳；测试中连杀用 `dungeonMgr.DefeatNPCNoLock`（直接结算目标），`Kill` 会把玩家置 COMBAT 导致下一杀 busy_state；`Kill` 同时是破招判定入口。

## 5. 属性系统

**两层公式，实际生效的是第二层**：`NormalizeAttributes`（存档兜底）≠ `effectiveAttributesForPlayerNoLock`（真实：四维→衍生重组 + 技能加成 + 装备加成 + 虚弱减半）。NPC 不走此链路（模板直读）。

- **改数值**：先改 formulas.md（含代码对照表定位），再动 `model/attributes.go` / `runtime_manager_effective.go` / `model/skill_progression.go`。
- **注意**：武学加成统一使用`*_per_100_levels`（见`applyLeveledBonus`）；ADVANCED需`requirements.prerequisite`与固定位置，满足资格并装备后才生效。新增字段须同步结算、文案和静态校验；未知字段会阻止资源加载。

## 6. 技能与学习系统

基础技能（世界NPC免费教入门）+命名武学（type ADVANCED）。自主修炼进入LEARNING状态，每10秒投入最多10潜能，悟真篇可增加预算；同次预算可连续升级。基本功受经验上限约束，命名武学上限为min(经验上限, 当前根基等级)。升级必然成功、无随机，无老师永久等级上限。

- **加基础技能**：skills.json + 老师 NPC 的 `skills` 映射。**武学总谱（流派/段位/老师/秘籍来源）见 [martial_arts_curriculum.md](design/martial_arts_curriculum.md)**——新增任何武学前先对谱，避免破坏段位覆盖设计。
- **新增武学**：skills.json声明type/固定rarity/slot/weaponType、根基、teacherId或sourceDungeon、manualId；完整秘本绑定不可售，来源副本满结算投放。统一资格覆盖初学/修炼/装备/属性，静态校验引用闭环，数量不锁死38。
- **传承与自修**：老师免费入门，后续PracticeSkill在安全公共房间每10秒按预算修炼，无老师永久上限。完整秘本失败不扣；五位置互斥，战斗须匹配主武器。
- **武学装备规则**：`Player.ActiveArts`为internal/movement/guard/combat/study五个互斥位置，每处择一。指令`equip_skill {skillId}`、`unequip_skill {slot}`；基本功自动参与结算，combat位置的基本功及命名武学均须匹配当前主武器。换装不回复气血或内力。
- **注意**：经验曲线因子与潜能档位是全局常量（`model/skill_progression.go`），改动即影响所有技能成长，必须先过 formulas.md §3。

## 7. 物品与背包

items.json：usable/effects（recoverHp回复、learnSkill参悟）、equipmentSlot、weaponType、firstClaim、sellable/bind/value。暗器并入WEAPON，工具回背包；实际加成字段为effects。恢复品战斗可用并共享10秒冷却；秘本须空闲。

- **注意**：删除物品前先全库搜索引用（副本 rewards、商店 shopItems、技能 requirements、剧情 giveItems）——孤儿物品由 gamedata 测试拦截；任务物品不可出售。

## 8. 商店（交易）

世界 NPC 商店：模板 `shopItems`（物品 ID 列表）。**运行时库存按 NPC 实例隔离**（`staticNPCStateNoLock`）——改模板不影响已生成的库存，也不会被玩家买空污染模板。定价：买=value，卖=⌊value/2⌋（任务物品不可卖）。

- **注意**：当前**没有玩家间交易系统**；银两出口以商店消耗品/基础装备为主，大额出口（修理/马车）是 economy_design.md §4 预留项。

## 9. 状态与恢复

玩家状态机：IDLE / COMBAT / MEDITATING(打坐) / HEALING(疗伤) / LEARNING / GATHERING(30s) / WORKING(60s) / DEAD。ProcessStatusTick（1s）驱动恢复与倒计时；Stop 中断并给出各自文案。恢复数值表见 formulas.md §5。

- **注意**：新增忙碌状态必须同时处理：checkExitRequirement/行动前置检查（后端 busy 拒绝）、前端 `useEntityActions.js` 的 stateNames 与 checkCanAct、Stop 的分支文案。

## 10. 对话系统

静态 NPC 对话：`dialogues.json`（id/npcId/title/opening/options），`Talk` 指令逐句推进；副本剧情对话见 §3.4（dungeon_dialogues.json）。NPC 模板的 `dialogueId` 指向 dialogues.json 条目。

## 11. 任务系统

`quests.json`：giver/turnIn NPC + objectives（当前实现为 **flag 型目标**：玩家持有某旗标即完成，如"听说书人传闻"）+ rewards（经验/潜能/钱/物品/声望）。设计原则：任务是一次性大额奖励与剧情钩子，**不做可重复刷源**（economy_design.md §1.2）。

- **加任务**：quests.json + 触发旗标的写入点（通常是房间动作或对话）+ 交付 NPC。

## 12. 势力与声望

`factions.json`（id/name/description/category/visible/order）。玩家有 Reputations 映射；副本结算可发 `reputations`。隐藏势力（副本专属）visible:false，只为 NPC 归属服务。

## 13. 聊天

公开聊天通道（type "chat"，subtype "public"），经 app 层广播。前端 `useChatState.js`。

## 14. 事件推送

服务端主动事件：`player_entered` / `player_left` / `player_status_changed` / `room_update` / `room_message` 等。前端 handleMessage 分发（GameView.vue）。**新玩法若需要他人可见的状态变化，走事件推送而不是让前端轮询**。

## 15. 持久化

全内存运行；30s 自动落盘 + 退出 flush；tmp+rename 原子写；损坏存档隔离为 `.corrupt` 并兜底；加载时 IDLE 且 HP≤0 的玩家恢复。动态数据在 `data/servers/server1/`（.gitignore 忽略）。

- **注意**：新增玩家字段必须同时改 storage 的 diskPlayer 序列化（读写两侧）与 `TestClonePlayersMapPreservesAllFields` 的字段数断言；**测试一律用临时 MUD_DATA_DIR + 独立端口**。
- **部署保留存档**：`deploy.bat` 每次构建最新程序，上传校验后再停服备份并切换；失败回滚程序和资源，存档保持原位。具体命令和备份位置见 [DEVELOPMENT.md](DEVELOPMENT.md#后端部署windows--linux)。称号 `titles.json` 与其他静态资源共用运行时目录解析，支持 `MUD_RESOURCES_DIR` 和跨平台构建。

---

## 通信协议速查（前后端联调）

- 请求 `{type, subtype, id, args}`：type ∈ `system`（账号）/ **`command`（游戏指令）** / `chat`。
- 响应 `{type, subtype, flag, results, logs}`：**flag 布尔即成败**，subtype 回显命令名；game 命令的响应 type 也是 `command`。
- 新指令全链路：`protocol`（如需）→ `app/command_registry.go` + handler → `engine/workflow*.go` → `manager` 方法 → 前端 `game/messageText.js`（错误码→文案）+ composable + GameView 装配。

### 当前全部游戏指令

| type | subtype | 功能 |
|---|---|---|
| system | register/login/delete_character/create_character/select_character/reconnect | 账号与角色 |
| command | get_attributes/get_backpack/get_skills/get_equipment/get_worlds/get_dungeons/get_quests/get_factions/get_shop | 查询 |
| command | enter_world/enter_dungeon/leave_dungeon/go | 移动与世界 |
| command | buy/sell/discard_item/equip/unequip/use_item/enhance_item | 物品与装备 |
| command | learn/equip_skill/talk/choose_dialogue | 技能与对话 |
| command | heal/meditate/stop/do_action | 状态与恢复 |
| command | kill/spar/relive | 战斗 |
| chat | public | 聊天 |

## 环境注意事项（Windows 开发机）

- `go test -race` 不可用（ThreadSanitizer 分配失败，环境限制）。
- `git grep` 在本机对部分场景不可靠，验证历史内容用 `git show <rev>:<path> | grep` 或 pickaxe（`git log -S`）。
- 资源 JSON 行尾约定：`npcs.json` 为 CRLF，其余资源文件 LF——脚本批量改写时按文件保持，避免整文件 diff 噪音。
- shell 工作目录会在调用间残留：**每条命令显式 `cd` 到目标目录**。

## 2026-10-27内容机制约定

任务含38个规范项和3个aliasOf兼容定义，sourceDungeon决定副本委托开放；接取和交付必须绑定可达NPC。Room的searchFlag/searchMessage声明一次性线索，rumors声明本房传闻。gamedata加载会拒绝缺少目标来源、接交入口或不可达主动作。

对话两条路径共享一次性历史；repeatable仅用于有真实成本的服务，Talk不自动制作，ChooseDialogue先检查全部成本及任务资格，再提交副作用。前端日志明确选择服务。GetQuests返回完整目录与状态、地点、目标进度，旧别名不重复呈现或发奖。

Stop及常规LeaveDungeon不能取消死亡，战斗退出需烟尘粉；UseItem的房间变化须推送旧房离开、新房进入及当前快照。打坐/疗伤起手不恢复，完整10秒才结算。公共老师、商人和剧情任务人物不能击杀，合适的老师仍可切磋；切磋按有效MaxHP检查满血及20%认输线。强化先校验空闲与持有，网页确认成本，5级上限与费用不变，按各属性独立系数增加；旧强化可逐ID进行一次退料重置。烟尘粉四店无限供给，250铜/包；详见[本轮规格](design/smoke_enhancement_2026_10_28.md)。


## 2026-10-28快速完成与掉落修复

详见[权威规格](design/dungeon_quick_complete_2026_10_28.md)。quick_complete_dungeon要求选中角色、公共世界空闲、该关满通关旗标和持久化expectedSequence；正常掉落与快速结算共用manager/runtime_manager_dungeon_rewards.go。模板quickRewards只配置途中固定items/money，资源加载验证来源，禁止临时钥匙/付费配方。副本protected须同时在实例Kill后端校验。


2026-10-28审查修复：安全区保护玩家（包括拒绝强制切磋），区外离线不免战；NPC默认被动。交易数量1～10000整数，检查乘加溢出；回收20%，由服务端sellPrice统一。所有配方物品价值严格递减，9组装备通过item_aliases.json规范化并幂等保留/退还旧强化投入。40关标needs_depth而不移除；详见[本轮规格](design/audit_repairs_2026_10_28.md)。
