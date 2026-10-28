# 武功主动招式：具体实现规范

状态：**设计与实现约定，未写入运行代码、未部署**。核对基线为运行代码7677ac4及文档提交e0c6e2c，日期2026-10-29。本轮仅补充现有38门武功；完整数值见[84招提案](active_martial_moves_proposal.md)，开发步骤见[实施计划](../superpowers/plans/2026-10-29-active-martial-moves.md)。本页中的新增类型、字段、函数、文件都是拟实施接口，不能据此判断线上已有技能。

## 1. 边界和代码核对

一次点击立即结算，然后同时进入角色出招CD与本招技能CD。所有招式基础出招CD为4000ms，攻速只缩短这个动作间隔；每招固定技能CD为10～60秒。普攻仍每2秒结算，不被替换、暂停、推迟或重开计时。每门1～3招，学会时全部获得，不按等级解锁。

| 当前文件（路径相对仓库根） | 已有情况 | 实施边界 |
|---|---|---|
| `MudGame-server/internal/model/types.go` | SkillDef没有moves；Player没有招式CD；NPC的Skills用于教学 | 新增下属招式定义、玩家冷却状态；NPC另设combatArts |
| `MudGame-server/internal/model/attributes.go` | AttackSpeed为int，默认4 | 保留旧字段，增加运行时浮点有效出招攻速；不能将4.8截成4 |
| `MudGame-server/internal/gamedata/static_loader.go` | rawSkill→SkillDef显式逐字段映射 | 两端及深拷贝同时补充；不能只给模型加JSON标签 |
| `MudGame-server/internal/manager/runtime_manager_combat.go` | 三条战斗路径；先算双方打击再应用 | 提取统一动作结算，按行动顺序逐次计算、逐次应用，再核死亡/认负 |
| `MudGame-server/internal/manager/runtime_manager_player.go` | Kill、Spar建立交战；满血切磋检查 | 成功建立战斗时绑定服务端会话；主动不能另选目标 |
| `MudGame-server/internal/manager/runtime_manager_dungeon.go` | 撤离/销毁有直接清理TargetID的路径 | 统一清理本场效果及意图，保留CD |
| `MudGame-server/internal/manager/runtime_manager.go` | 全局mu保护状态；30秒自动保存，Close落盘 | 即时结算在锁内提交；网络发送在锁外；沿用保存语义 |
| `MudGame-server/internal/storage/store.go` | diskPlayer、LoadPlayers、SavePlayers显式映射 | 三处增加可选combatMoves字段，旧档零值兼容 |
| `MudGame-server/internal/manager/runtime_manager_persistence.go` | clonePlayersMap显式复制字段 | 深拷贝冷却map，更新全字段测试 |
| `MudGame-server/internal/app/server.go` | 1秒状态tick、2秒战斗tick | 保留两者，单增100ms NPC意图调度；不在100ms结算普攻/流血 |
| `MudGame/src/composables/useCommandQueue.js` | 所有command超限后排队 | cast_combat_move必须限流后立即发送或立即拒绝，不能进入等待队列 |
| `MudGame/src/views/GameView.vue` | 统一装配composable、接收WS消息 | 注入招式状态和发送函数；composable间不相互导入 |

当前有三个容易遗漏的实现点：双方伤害预计算会使先手施加的削弱不能影响本轮后手；AttackSpeed的整数格式无法表达20%加速；前端通用指令队列会让旧点击在数秒后施放。这三处要随本功能一起处理，不能只增加按钮。

不修改38个武功ID、物品/地图/NPC实体集合、MartialVersion=2或原入门条件。不创建新的秘籍卷；不改变武器装备、普通用药、烟尘粉、快速完成的独立规则。

## 2. 内容模型、稳定ID与配置校验

招式仅作为原`resources/skills.json`中武功的moves子数组；稳定ID为`武功ID/m01`至`/m03`。编号在本页附录冻结，不随排序、中文改名或按钮位置改变。删除后保留墓碑映射，不能复用旧编号；扩展时仍遵守一门最多3招，本次无新增武功。

建议模型放入新增的`internal/model/combat_moves.go`，现有SkillDef新增`Moves []CombatMoveDef`，Player新增`CombatMoves CombatMoveState`，NPC新增`CombatArts []NPCCombatArt`。以下为核心数据契约（JSON采用同名小驼峰字段）：

```go
type CooldownWindow struct { StartedAt, ReadyAt int64 }
type CombatMoveState struct {
    Sequence int64
    Action CooldownWindow
    Skills map[string]CooldownWindow
    Blocks map[string]CooldownWindow
}
type MoveCondition struct { Kind string; Value float64 }
type MoveEffect struct {
    Op, Target, Trigger, Scope, Group, Stat string
    Value, Cap, Power float64
    Charges, Ticks, DurationMs int
    Tags []string
}
type CombatMoveDef struct {
    ID, Name, Kind, Description string
    CooldownSec, ActionMs, MPCost, HitCount int
    Power, HitBonusPP, ArmorIgnore, ParryReduction float64
    Conditions []MoveCondition
    Effects []MoveEffect
}
type NPCCombatArt struct { ArtID string; Level int; MoveIDs []string }
```

Kind仅允许attack、support、debuff。直接攻击的Power是全动作D倍率，HitCount为1～3；support/debuff的HitCount与Power均为0。ParryReduction表达削减幅度（“招架概率乘0.5”存0.5，“乘0.7”存0.3），成长后最多削减70%。HitBonusPP用小数概率，如0.15，不存15；面板输出“15个百分点”。DurationMs、Charges、Ticks是固定期限，不乘成长系数。MPCost整数，消耗当前MP。

MoveEffect.Value为100级幅度，Cap为该项基础上限，Power用于反击/流血D倍率；唯一允许的Target为self/opponent。Trigger为cast/hit/incoming/damaged/missed/next_tick；Scope为active_attack/normal_or_active/direct_incoming/multi_active/dot/self，具体组合由下表白名单约束。Group决定同组取强，不是可任意执行的脚本。条件只允许self_hp_below、target_hp_at_most、recent_dodge、recent_active_damage、self_hp_cost；“惊雷趁隙”对敌剩余出招CD的判断是可选增伤条件，不是施放门槛。

| Op | 参数含义及合法触发 | 对应目录实例 |
|---|---|---|
| heal / hot | self、cast；Value为MaxHP比例，hot固定Ticks并从下一战斗tick开始 | 护息、水纳百川、生息归元 |
| next_active | self、cast；Stat为damage/hit_pp/crit_pp/armor_ignore/guard_pierce，Scope=active_attack | 察势、知隙、战神凝势、照见虚实 |
| guard | self、cast→direct_incoming；Value为减伤，Charges/DurationMs双限制 | 架挡、神意护体、金刚护体 |
| stat_mod | 自用cast或敌向hit；Stat为attack/defense/hit/dodge/parry/initiative/crit_pp；负值为削弱 | 挪步、雷回夺势、拿虎卸劲、压阵封门 |
| outgoing_reduce / vulnerability | outgoing_reduce作用对手normal_or_active；vulnerability作用自己direct_incoming | 截脉封劲、血战冲阵、天刀一线 |
| counter | self、incoming；仅对手normal_or_active触发，Tags标明damaged或missed，Power为反击倍率 | 铁壁迎击、回马枪、虚实换步 |
| bleed / lifesteal | opponent或self、hit；基于整招实际伤害；bleed总预算另按Power×D快照 | 雨落无声、魔极蚀心、血刃回身 |
| cleanse / immunity | self、cast；Tags限定可清除/免疫组；DurationMs限定免疫 | 澄心明鉴、定波守心、清浊守一 |
| mp_discount | self、cast；不优惠本招，Charges=2，Value随等级成长，上限35% | 聚气归元 |
| haste | self、cast；仅改变有效出招攻速，DurationMs=8000 | 步罡抢势 |
| conditional_power / conditional_ignore | cast检查HP阈值或目标出招剩余时间；只改变本次攻击 | 浴血斩、修罗追命、惊雷趁隙 |
| dispel_bonus / strip_guard_charge | opponent、hit；移除限定增益/额外扣1次护体计数 | 镇步伏形、凝虚破障 |
| paid_mp_refund | self、damaged；Scope=normal_or_active，带本招实付费用账本 | 守中蓄劲 |
| delayed_damage_heal | self、damaged；Scope=active_attack，按整招实扣HP在下一tick恢复 | 镜湖倒影 |
| recent_damage_heal | self、cast；使用最近同对手主动伤害记录 | 踏雪折影 |

`internal/gamedata/combat_move_validation.go`实现`ValidateCombatMoves(skills map[string]*model.SkillDef) error`；实际容器若沿用现有值类型，统一按现有类型调整该签名与调用端，不能创建第二份技能表。校验ID归属和唯一性、1～3招、CD10～60、基础动作4000、MP>0、有限浮点数、倍率/比例/段数界限及所有Op/Trigger/Scope组合。拒绝未知字段和未知枚举；没有运行时eval。支持无moves的旧资源整包启动；有moves的发布包必须覆盖全部现有武功，不能混入半份目录。发布目录测试固定38/84，加载器不能把游戏未来总武功数硬编码成38。

示例：惊雷趁隙的附加条件，不改变技能CD或目标CD：

```json
{"id":"thunder_sword/m03","name":"惊雷趁隙","kind":"attack","cooldownSec":35,"actionMs":4000,"mpCost":18,"hitCount":1,"power":1.4,"effects":[{"op":"conditional_ignore","target":"opponent","trigger":"cast","scope":"active_attack","group":"armor_ignore","value":0.35,"durationMs":1000,"tags":["target_action_remaining_gte"]}]}
```

这里durationMs=1000是条件阈值，conditional_ignore不创建持续状态；加载器按该Op校验字段语义。其他持续Op的DurationMs才是存活期限。纯攻击招式无特殊条件时Conditions/Effects可为空；实际配置必须与附录逐行效果一致。

## 3. 运行时状态与战斗身份

新增`internal/manager/runtime_manager_combat_moves.go`负责入口，`runtime_manager_combat_effects.go`负责效果，`runtime_manager_combat_sessions.go`负责战斗绑定，`runtime_manager_combat_ai.go`负责NPC意图。都由现有RuntimeManager.mu保护，不另引入跨锁相反顺序。

```go
type ActorKey struct { Kind, InstanceID, EntityID string }
type BattleBinding struct {
    ID string
    Self, Target ActorKey
    RoomID, Mode string
    StartedAt int64
    AutoRounds int
}
type RecentCombatEvent struct {
    BattleID string
    Source ActorKey
    At int64
    ActualDamage int
}
```

ActorKey.Kind为player/static_npc/dungeon_npc；公共NPC用实际房间及运行时实体，副本NPC必须带实例ID和实例实体ID，不能把模板ID当运行时实体。每个角色绑定一个当前目标；目标改变必须生成新的不可猜BattleID，清除旧目标相关临时效果/最近事件/待回疗，保留角色CD。公共NPC自身CD与效果按实体共享，不按每个攻击者复制一份；一次意图只绑定一个合法对手。

`Kill`/`Spar`成功进入交战后调用`beginCombatBindingNoLock`。PvP双方有各自绑定，允许现有多人攻击关系；主动只打本人的当前合法目标。目标撤离、死亡、认负、烟尘粉脱离、离开副本、实例销毁及`resolveOtherCombatantsTargetingNoLock`自动转目标，都经`endCombatBindingNoLock`或重建绑定。不能仅在按钮请求处检查State。离线不清战斗，原可被攻击/自动战斗行为继续。

临时效果只在服务端内存：稳定effectID、来源、目标、battleID、施放等级快照、创建/到期毫秒、剩余次数/tick、已付MP/已返还MP、下一tick预算。effectID由角色序号+招式ID+效果索引生成。次数限制与时间限制取先到者；战斗结束一起清理正负效果。不将临时效果回写Derived基础属性。

重启时旧BattleID全部作废，清空临时效果与NPC意图，清除战斗TargetID/CombatType并将仍COMBAT者归IDLE（DEAD保持）；保留HP/MP、序号与真实时间CD，不执行Spar结束补满血路径。这里必须单独区分“重启恢复”与既有切磋正常认负处理。

## 4. 双CD、攻速、保存与进度窗口

基准AttackSpeed=4。`effectiveActionSpeed = normalizedBaseSpeed × (1+hastePct)`保留float64；没有增益时4，20%加速时4.8。`actionMs=max(1000,ceil(baseActionMs/(effectiveActionSpeed/4)))`。本轮所有baseActionMs=4000，1000ms是候选下限；无新增攻速装备或强化系数。自用haste在本次动作时长快照前生效，之后失去增益不倒改已开始的CD。技能CD永远使用配置秒数，普攻仍2秒。

新增`internal/model/combat_move_clock.go`提供以下纯函数；以Unix毫秒为保存格式，测试注入固定时钟，不能在测试中真实等待：

```go
func ActionDurationMs(baseMs int, effectiveSpeed float64) int64 {
    if effectiveSpeed <= 0 || math.IsNaN(effectiveSpeed) || math.IsInf(effectiveSpeed, 0) {
        effectiveSpeed = 4
    }
    return int64(math.Max(1000, math.Ceil(float64(baseMs)*4/effectiveSpeed)))
}
func MergeMoveBlock(old CooldownWindow, action, skill CooldownWindow, now int64) CooldownWindow {
    end := max(action.ReadyAt, skill.ReadyAt)
    if end <= now { return CooldownWindow{} }
    if old.ReadyAt > now && old.StartedAt <= now {
        return CooldownWindow{StartedAt:old.StartedAt, ReadyAt:end}
    }
    return CooldownWindow{StartedAt:now, ReadyAt:end}
}
```

文件import math，项目Go版本支持max；外部clock仅由manager的`combatNow func() time.Time`注入，缺省time.Now。施放成功记录Action/Skills[id]，并立即更新本人**所有已有招式**的Blocks（至多84项），包括当前未装备的，保证换装/重连不重置视觉窗口。没有在某技能CD内的其他按钮也因actionReadyAt进入阻塞；连续阻塞时保持原StartedAt。普通查询不能重新从now开始。清理过期Skills/Blocks只在无阻塞后进行，序号不重置。

| 当前A按钮 | 新施放B | A的结果 |
|---|---|---|
| A本招还剩12秒 | 新出招CD4秒 | A仍按原结束时刻结束，不回满、不改技能CD |
| A本招还剩2秒 | 新出招CD4秒 | A实际解锁延后到4秒；保留原窗口起点，显示新真实限制 |
| A已经可用 | 新出招CD4秒 | A新窗口起点为B施放时刻，条满后线性减少 |

进度`fraction=clamp((blockedUntil-now)/(blockedUntil-blockedSince),0,1)`，blockedUntil是max而非CD相加。新实际限制延长窗口时，条可上升；详情说明是角色再次出招，不是A技能CD重置。所有deadline由服务器决定。

Player.CombatMoves完整进入diskPlayer、LoadPlayers、SavePlayers和clonePlayersMap，Skills/Blocks深拷贝；缺字段读零值。保存复用现有30秒自动落盘及正常Close落盘，不声称异常断电零丢失。未来若要求每招崩溃后也严格只结算一次，需要另加事务日志；当前重启旧BattleID失效，不能用旧请求续打，但尚未保存的费用/CD和进度同样遵守现有存档恢复粒度。

加载时验证非负序号、合法ID、合理窗口顺序和毫秒范围，过期项清理，未知合法旧招式ID保留CD到期后移除；有明确改名才通过别名映射迁移，不映射到按钮位置。旧版本回滚可能丢弃新增可选字段，必须在发布记录中声明：回滚后再升级不承诺恢复其间未保留的CD；不能把普通JSON兼容说成无损往返。

## 5. 原子入口、协议与推送

统一接口放在model；engine.RuntimeRepository新增：

```go
GetCombatMoves(playerID string) (model.CombatMovesView, error)
CastCombatMove(playerID, moveID, battleID string, expectedSequence int64) (model.CombatMoveResult, error)
ProcessCombatMovesTick() (model.CombatTickResult, error)
```

`CombatMovesView`字段：capabilities（包含combat_moves_v1）、serverEpoch（本进程启动随机串）、revision（该玩家视图变化单调计数）、serverTime、battleId、sequence、mp、effectiveActionSpeed、actionStartedAt/actionReadyAt/actionDurationMs、moves。MoveView逐招含id/name/artId/slot/level、实际效果和费用、skillStartedAt/skillReadyAt、blockedSince/blockedUntil、unavailableReason。sequence只成功施放才加1；revision在伤害、效果变化、意图、失效目标等视图变化时也递增。非战斗可查已装备招式详情，battleId为空且不可施放。

`CombatMoveResult`含view、effects（model.CombatTickResult）、code、appliedSequence；失败同样能附view，不要让普通error返回导致权威状态丢失。model.CombatTickResult增加私有视图集合`CombatMoves []CombatMovesUpdate`，每项playerId+view，只定向发本人；不得混入公共房间payload。

`internal/engine/workflow_combat_moves.go`只做身份后的工作流包装；`internal/app/handlers_combat_moves.go`使用s.requireSelectedPlayer，忽略客户端声明的其他playerId。`internal/app/command_registry.go`注册get_combat_moves与cast_combat_move。沿用protocol.CommandMessage、GameResult，不另开WebSocket。

```json
{"type":"command","subtype":"cast_combat_move","args":{"moveId":"thunder_sword/m03","battleId":"server-generated-token","expectedSequence":7}}
```

只接受这三个参数，严格拒绝额外的targetId、伤害、攻速、费用或CD参数；expectedSequence为非负JSON安全整数。通过现有会话选角身份确定角色。锁内流程：

1. 取得now，解析当前角色/合法目标/绑定和room；拒绝过期battleId、安全区、禁止攻击NPC、已结束等情况。自己与目标状态即使在上次查询后变化也重核。
2. expectedSequence必须等于当前sequence；较旧返回stale_sequence及最新view，不执行、不重播旧伤害，不需要无限缓存历史回包。相同序号不同招式并发仅一次成功；首次失败不推进序号，后续新点击仍可用该序号。
3. 核对已学、装备/基本功回退、武器、既有传承要求、MP、双CD、特殊HP/最近事件条件。先建立只读计划，确认纯治疗/护体是否有收益。失败不改费用、随机数流、次数、冷却或序号。
4. 原子扣实付MP/合法自损，消耗一次优惠；按§6立即执行本招，创建效果。命中失败属于施放成功，仍扣费进CD。
5. 计算攻速快照，记录双方相关状态、本人Action及本招Skills、Blocks和sequence+1、dirty标记。即使本招刚结束战斗也先保留该次成功施放的CD；清理战斗效果不清冷却。
6. 构建不可变回包/房间变化/私有视图后释放mu；app统一发送effects和view。网络异常不回滚已结算动作，重连查状态。锁内不sleep、不等待出招CD、不广播I/O。

get_combat_moves、施放成功/失败、战斗tick、用药、换装、战斗状态变化都可更新私有view。公共仅披露本房可观察的HP/动作/防护、当前对手actionReadyAt及NPC公开intent；不会下发对手整套技能CD、技能表、MP私密明细或sequence。惊雷趁隙的条件据服务器目标状态判断，客户端展示只用于提示。

所有时序用同一now和manager锁串行化。1秒主动、2/4/6秒普攻必须各自执行；不将主动并入下一战斗tick。角色A的出招CD不影响角色B，mu只用于短暂计算，不持锁等待4秒。

## 6. 统一伤害与效果顺序

新增`internal/manager/runtime_manager_combat_action.go`，提供`resolveCombatActionNoLock(action CombatAction) CombatActionOutcome`。CombatAction包含actor/target的ActorKey、bindingID、kind（auto/move/counter/dot/self）、moveID、hitCount、总倍率、procDepth、now；Outcome包含每段命中/招架/实扣HP、总实扣HP、结束原因、待发送日志。由结构化命中结果区分闪避，不从中文日志解析。

三条原战斗路径统一取实体适配器，负责读写实际HP/MP、有效属性与结束规则；击杀奖励和副本进度仍调用既有resolveKillEndNoLock、defeatDungeonNPCNoLock等入口，只执行一次。ProcessCombatTick仍在原2秒时点运行，按原先手排序后，每次出手前读取最新状态并结算；不能先计算后手伤害再让先手减益“只影响下轮”。

每个直接攻击动作按如下顺序：

1. 检查双方仍在同一合法战斗；取属性快照、适用的下次主动强化/敌向削弱/来袭护体。按原装备、被动和副本弱点得到防御基准，再加主动削弱；技能防御削弱与忽略合计不能超过基准50%。
2. 计算`D=max(2,attack-floor(defense/3)+modeBonus)`，KILL修正14、SPAR修正8，倍率按`S(L)=0.8+0.4L/(L+100)`成长。整动作先算整数预算，再分配到HitCount，余数按段序分配。例如总5拆3段为2/2/1，不是三次独立保底D。
3. 每段按原命中/波动/暴击/招架顺序计算，命中15%～95%，暴击总计≤25%；本招额外命中最多25个百分点。零预算段不追加伤害，正预算命中最低1。各段不重复整份防御或触发整份附加伤害。
4. 防护减伤取最强，先扣护势穿透（最高30%）：40%护体×(1-25%)=30%。再与攻击方outgoing_reduce等主动乘区合并，主动最终减伤最多65%；易伤是独立负面代价。次数型护体对dot不生效。原装备防御与招架不受此主动减伤上限替代。
5. 切磋为整动作累计设置floor(目标MaxHP/5)预算，不能每段重置；每次HP变化后检查1HP、20%认负/副本25%认负或0HP死亡，达到即终止所有剩余动作。自损先检查支付后不致死/不触认负线，否则整招拒绝。
6. 来袭计数整个动作扣一次；全部落空仍消耗来袭防护/闪避。下N次主动强化也只扣一次，落空仍消耗；新建效果不在施放当次倒扣。敌向“下N次攻击”削弱适用auto/move；“下N次承受攻击”防御/招架削弱适用直接来袭，命中或落空按一次动作消耗。
7. 若总实扣HP>0，执行一次命中后减益、流血/吸血、驱散。自用防护与自身破绽无论命中仍生效；敌向纯减益单独掷命中，无伤害、无暴击/招架，不消耗next_active增伤，也不触发来袭防护/反击。死亡/认负后的敌向附加效果不继续挂。
8. 更新recent_dodge/recent_active_damage（同一动作聚合）；再按触发过滤器处理至多一次反应。反击需本人仍能出手且战斗未结束；反击procDepth=1，不触发反击、吸血、MP返还或镜湖回疗。直接吸血作为原动作结算的一部分，本人仍存活时可取击败目标前实扣伤害回复；不能在本人死亡/认负后补回。

多段不能触发多份counter/bleed/refund；反击使用触发时有效攻防与原姿态等级，不使用next_active强化。防护减伤与其独立反击触发分开：护体同组取强，较弱防护不覆盖/续期，但本招独立反击资格仍可存在。合格反击取本次伤害预算最大者，其他反击资格也消耗这次对应来袭计数，不能积累成多段反击。

同组增益按当前有效幅度取强，等幅保留旧项，不刷新期限/次数；较弱不排队等待强项失效。躲闪增益最高45%，其他同类属性变化最高35%；暴击与命中概率另按概率上限。先手效果持续随后两个原战斗轮，在该轮末消退，不改变已确定的本轮顺序。其余未标明期限的次数效果最长12秒。

持续伤害/治疗预算施放时快照后整数拆分；下一2秒战斗tick开始，tick去重，最多配置次数。战斗tick内部先核失效/到期，再处理先前挂起的持续效果和回疗、逐次检查结束，然后进行仍合法的普攻；本tick新建的持续效果要到下tick才执行。不得在100ms意图调度或查询时多执行。dot不消耗来袭护体、不触发反应，仍遵守切磋/认负线。

## 7. 十二个补充招式的可执行规则

本次38门由72招补为84招；十二门由2招增为3招，均占既有m03编号。目的为增加反应与组合选择，不按品阶机械补满。

| 稳定ID／招式 | 新增判断与效果引擎组合 | 明确不触发的情况 |
|---|---|---|
| thunder_sword/m03 惊雷趁隙 | 1.4D，目标actionReadyAt-now≥1000时本招忽略35%防御 | 只看当前目标；不改变其CD；技能CD35s/MP18 |
| hunyuan_neigong/m03 清浊守一 | cleanse(bleed,1)+immunity(vulnerability,4000ms) | 无流血仍可获得免疫；不免其他削弱；45s/18 |
| talang_jue/m03 踏浪穿隙 | recent_dodge条件+next_active(hit_pp=.15,damage=.10,1次,12s) | 只认同目标auto/move命中落空；20s/10 |
| fuhu_jin/m03 镇步伏形 | 1.0D+dispel_bonus(initiative,haste,1项) | 不移除装备，不倒改现有CD；35s/16 |
| tiebi_gong/m03 守中蓄劲 | guard(.20,2次,12s)+paid_mp_refund | 不接受counter/dot/self；40s/16 |
| wuzhen_pian/m03 照见虚实 | next_active(guard_pierce=.25,1次,12s) | 不穿透被动/原防御/免疫，最高.30；35s/16 |
| taiyi_jianfa/m03 守一归锋 | cleanse(vulnerability,1)+next_active(damage=.15,hit_pp=.10,1次,12s) | 普攻不使用也不消耗强化；40s/18 |
| dongting_xinf/m03 镜湖倒影 | guard(.20,1次,12s)+delayed_damage_heal(.15,cap=.05) | 只认move，最多一次，下一tick回疗；40s/20 |
| bawang_qiang/m03 压阵封门 | 1.0D+stat_mod(crit_pp=-.10,normal_or_active,2次,12s) | 最低0，不改变原25%上限；35s/20 |
| lieshi_jian/m03 沉锏定势 | guard(.50,multi_active,1次,8000ms) | auto/counter/dot不享受也不消耗；45s/24 |
| poxu_zhi/m03 凝虚破障 | 1.2D+strip_guard_charge(1) | 先自然扣次再额外扣次，无护体只伤害；50s/30 |
| taxue_wuhen/m03 踏雪折影 | recent_active_damage条件+recent_damage_heal(.20,cap=.05)+dodge(.25,1次,12s) | 只取最新同对手一笔实际伤害，多段合一；40s/20 |

所有正面幅度随S成长，条件阈值/次数/期限固定，通用上限不成长。特别规则：

- 最近事件有效条件：event.BattleID等于当前binding.ID，event.Source等于当前target，0≤now-event.At<4000。recent_dodge只记录命中检定落空（招架不等于闪避）；recent_active_damage只记录对手kind=move且ActualDamage>0的整动作总实扣血。不能由客户端上传。更换目标/结束立即丢弃。成功使用时消耗对应记录，防止将来新增缩CD后重复兑换同一事件。
- MP返还：每个合格实扣血动作最多floor(4S)，累计最多min(floor(8S),floor(paidMP/2))，还受当前MP缺口限制；只返实得量，未返部分不在解除/死亡时补发。100级实付16最多返8；折扣后实付12最多返6；实付11最多返5。两个受击窗口分别扣次，反击虽可消耗护体但不返MP。任何等级均不能净产内力。
- 镜湖回疗：guard对应来袭消耗时捕获同一move总实扣血，预算min(floor(damage×.15S),floor(MaxHP×.05S))，有缺血且正比例最少1。先绑定目标/战斗，到下一tick核存活、同场且未认负后再恢复；战斗结束取消，不能变成复活。该防护被普通攻击消耗时没有回疗；此差异是玩家使用时机的代价。
- 踏雪恢复同样先用实际旧伤乘比例，再用本人MaxHP上限，最终受当前缺口限制。免疫只阻止新增相应减益，不清旧项。纯护势穿透在对方没有护体时仍可蓄势，12秒后未用即失效。
- 移除最强增益按幅度、稳定effectID排序；消护体按当前减伤幅度、剩余次数降序、稳定effectID升序。澄心清除按提案明确的负面优先级，不能用map迭代随机选项。

## 8. 三个NPC试点与调度

只给现有td_song_que、lj_pang_ban、db_zhang_xiu配置combatArts；具体招式、MP和HP优先级见提案§7。level=100只算招式，不重复加武功被动；保留原弱点/认负/奖励。所有NPC使用实体共享MP/CD，不按玩家复制。教学Skills完全不变。

新增`ProcessCombatMovesTick`由100ms ticker触发：仅处理合法NPC意图，不执行玩家排队招式、普攻、dot/hot。开战后AutoRounds≥1且没有现存意图、两CD就绪、MP足够时，按配置优先级选招；稳定绑定一个目标，发布intent={moveId,name,targetId,announcedAt,releaseAt:now+2000}。2秒后第一调度点立即调用同一施放引擎；默认100ms调度精度，不保证零误差。预告期间普攻继续。

意图公开后不换招、不换目标追击；目标失效、自己认负/死亡、两CD/MP/条件不满足时取消，不扣费不启动CD。NPC不能根据玩家未公开按钮请求决策；不提前抽伤害随机数。多人围攻时沿用现有合法交战目标规则，确定性选当前目标（没有则稳定ActorKey顺序的有效攻击者），不是每100ms重新找血少者。断线不取消合法交战，退出实例则取消。只在意图创建/取消/结算时推送，空100ms tick不广播。

## 9. 前端接入与真实冷却显示

新增`MudGame/src/composables/useCombatMoves.js`，签名：`useCombatMoves({ sendImmediateCommand, addLog, playerId, monotonicNow })`。暴露view、pending、applySnapshot(snapshot)、castMove(moveId)、reset()、dispose()。只依赖注入对象和Vue，不导入别的composable；由GameView装配。新增`src/game/combatMoveClock.js`导出`cooldownFraction(window,now)`纯函数，新增`src/components/GameCombatMovesPanel.vue`接收view/pending/now并emit('cast',id)。GameActionPanel容纳该组件，GameView负责传参与处理结果。

现有useCommandQueue新增`sendImmediateCommand`：与原command共享15条/2秒额度，额度用尽立即返回false并提示“操作过快，请稍后重试”；成功立即调用原rawSendGameCommand并返回true。**cast不进commandQueue，不设置等待flush的timer**。原其他指令队列照常。发送失败即解除pending，不扣MP、不启动CD；发出后等权威回包，禁止多次pending；断线/换角色立即清pending和视图，下次重连查询，不重发旧cast。

view的(serverEpoch,revision)用于忽略旧包；跨epoch仅在已确认的新WS连接上接受并reset，不能让旧socket包触发反向切换。serverTime校时：收到包时记录本地performance.now，之后`displayNow=serverTime+(performance.now()-receivedMono)`；下一包仅在revision不旧时修正，不从Date.now自行授权施放。网络延迟可能使显示略保守，服务器最后判定；本轮不引入预测伤害或预扣费用。

```js
export function cooldownFraction(window, now) {
  if (!window || window.readyAt <= now || window.readyAt <= window.startedAt) return 0
  return Math.min(1, Math.max(0, (window.readyAt - now) / (window.readyAt - window.startedAt)))
}
```

按钮以绝对定位遮罩显示`width: fraction*100%`、opacity:0.20、pointer-events:none，底部文字始终清晰；动画用requestAnimationFrame或≤100ms显示tick，不能以整数秒阶梯更新宽度。倒计时≤10秒向上取到0.1秒，>10秒向上取整数，未结束不显示0。详情分列两种CD；缺MP/条件不满足单独说明，不拿这些理由无限延长冷却遮罩。冷却归零后有其他禁用原因仍不可点。

服务器未声明combat_moves_v1时隐藏可操作栏；不为旧服生成假技能。旧客户端不请求技能也能继续普攻。私有视图与公共NPC意图通过现有事件分发，不新开连接，不在浏览器控制攻击tick。展开详情展示修炼等级后的实际数值及基础4秒/实际出招时长，不能把4秒显示成施放前读条。

## 10. 验证矩阵与数值门槛

| 测试边界 | 确定输入与必须断言 |
|---|---|
| 目录 | 38个原ID完全相等，84个唯一moveID；各门1～3，基础4s，CD10～60，MP正值，无等级解锁字段；三个NPC教学表不变 |
| 成长/上限 | L=1/20/100/200幅度单调；招架削减方向正确，guard_pierce≤30%、总主动减伤≤65%、defense ignore≤50%、crit≤25% |
| 时钟 | effectiveSpeed=4/4.8/5/8→4000/3334/3200/2000ms；极大值下限1000；技能CD不变 |
| 双CD | t=1000施放A(CD20s)，action到5000，A到21000；t=5000施放B(CD10s)→action到9000，A仍21000；A/B同时点只一次成功 |
| 原子/重放 | 成功扣一次MP、seq+1；相同seq重放无伤害/费用/新CD；失败无随机数消耗；断线重连不重发；同一时刻两角色互不阻塞各自CD |
| 普攻并行 | 预定2000/4000/6000三次auto不变，1000主动额外即时发生；4000先手减益影响本轮后手；击败则合法取消之后攻击 |
| 进度 | A剩12s时B出招4s不改变A截止；A剩2s时延至4s但skill截止不变；t=2000窗口[1000,21000]剩.95，重连不满条 |
| 触发域 | 察势/知隙被auto、counter、dot均不消耗；下次move消耗一次；所有落空仍扣有效来袭次数；多段整体扣一次 |
| 三路径/保护 | PvP含离线、公用NPC、副本NPC共用结算；安全区玩家、保护NPC、已死/认负目标拒绝；满血开切磋仍执行 |
| 多段/切磋 | 整招总预算5拆3段=2/2/1；SPAR一招≤MaxHP/5，20%与NPC25%逐段停止，HP不低于1；dot/counter/self也不过停止线 |
| 新12招 | 4秒窗口3999ms通过/4000ms拒绝、异目标拒绝；20%优惠后实付与MP返还上限，镜湖下一tick前认负取消；拆段只记录一次；消护体自然一次+额外一次 |
| AI | 至少一次auto后预告2秒，预告及出招CD内auto继续；MP耗尽只auto；取消意图不扣费；围攻共享CD，同模板不同实例互不共享 |
| 保存 | tempfile里旧档空字段加载；新档正常Close保存/重启，CD和seq不丢、HP/MP不重置，临时效果/旧battle token清除；clone全字段深拷贝 |
| 前端/协议 | 即时指令超额拒绝且不排队；重复/乱序视图不倒退；断线清pending；单遮罩20%不挡字，权限/MP提示正确，旧服务无假按钮 |

实现后新增84招表驱动测试，对每招至少正常施放、缺MP、技能CD、出招CD、非法目标、核心实际HP/效果次数断言；不只检查日志。特殊效果用上表专测，所有随机判定使用固定种子/注入结果。存档用t.TempDir；HTTP/WS用独立端口和独立MUD_DATA_DIR，禁止触碰正式存档。

数值仍是候选。由于主动是普攻以外的额外收益，实施必须比较同经验/潜能/装备预算的9类战斗兵器配置，在L=20/100/200下跑固定100个种子，每局上限120秒，三策略为仅普攻、按可用按钮顺序、按受击/护体/弱点选择。记录胜率、P50/P90时长、auto时间戳、主动伤害/治疗/减伤、MP净消耗与卡死局。对三个试点用可在其之前正常获取的武功/装备，不能使用本首领掉落物作前置。

发布阻断：任何超界/循环/永久僵局、没有预告的开场特殊伤害、任一基准可通关配置加入主动后三试点胜率下降超过20个百分点或降至50%以下、无响应窗口内主动组合普遍击倒同预算满血目标。数值阈值是本设计验收门槛，不代表已经测得的结果；失败先调整明示倍率/费用/CD，不暗加隐藏抗性。实验报告保存`docs/design/active_martial_moves_balance_results.md`，列种子与配置供复现；没有报告不能标记“平衡完成”。

## 11. 文件职责、发布与玩家说明

完整文件与测试任务见实施计划。`formulas.md`在开始改运行代码前写入获确认公式与冷却规则；该文件仍是已实施数值的权威入口，当前用链接标注候选而非伪造上线状态。后台完成后才为`resources/skills.json`增加实际moves，为三试点NPC增加combatArts。生成器读取真实moves输出玩家38门招式表，手写玩法页补充选择/进度/普攻并行说明；不能现在把84招写成已能玩。

不需武学V2重迁移，不加-MigrateMartialArts。打包必须包含匹配的程序/资源/前端；先BuildOnly、StageOnly，再按后续用户部署指令发布，保留原备份和正常停服落盘。回滚复用已有流程，已保存可选字段的兼容限制见§4。当前任务不进行部署或推送。

## 12. 本轮文档验收记录

2026-10-29（实施记录）：服务端与前端按本规范实施完成。`model/combat_moves.go`、`model/combat_move_clock.go`、`manager/runtime_manager_combat_{sessions,action,effects,moves,ai}.go`、`gamedata/combat_move_validation.go`、协议`get_combat_moves`/`cast_combat_move`、100ms意图调度、前端`useCombatMoves`/`GameCombatMovesPanel`与立即发送通道均已落地；84招配置按附录冻结ID写入`resources/skills.json`，三试点combatArts已配置。实施中按数据语义补充的两处枚举：`stat_mod`允许负值（削弱）、新增`parry_mult`（锁浪借势乘数）、`next_active`允许`missed`触发（逆浪回步）、命中加成允许-25pp～+25pp（乱披风）。Go全量与前端门禁通过；战斗平衡对比（三阶段三路线两策略）见[平衡报告](active_martial_moves_balance_results.md)，未宣布"平衡完成"。真实WS与浏览器验收及九路线全矩阵仍待后续。

## 历史记录

2026-10-29：仅做文档与纯公式检查，不执行运行代码实现。核对现有38个武功的ID/名称无增删；扩展为84招，品阶招式数13/21/27/10/13，全部费用为正、CD10～60秒、基础出招4秒；最大五位置配置15招。附录ID与目录逐行对应，不重号。检查成长/攻速/合并窗口及受折扣MP返还公式边界，并核查文档相对链接。候选战斗平衡、真实WS和页面尚待实施阶段验证。

2026-10-29：仅做文档与纯公式检查，不执行运行代码实现。核对现有38个武功的ID/名称无增删；扩展为84招，品阶招式数13/21/27/10/13，全部费用为正、CD10～60秒、基础出招4秒；最大五位置配置15招。附录ID与目录逐行对应，不重号。检查成长/攻速/合并窗口及受折扣MP返还公式边界，并核查文档相对链接。候选战斗平衡、真实WS和页面尚待实施阶段验证。

2026-10-29（补充）：落实附录84条冻结ID映射，逐条对照提案§5目录顺序与§7新增12招的m03编号，合计84条无重号、无遗漏；新建[实施计划](../superpowers/plans/2026-10-29-active-martial-moves.md)，其文件清单、接口签名与任务顺序均取自本页§1/§5/§8/§9/§11及提案§10/§11，未引入新机制或新数值。实施仍待用户确认；运行资源与代码未改动。

## 附录：冻结的84招ID

以下m01/m02/m03顺序即稳定编号，不因前端显示顺序改变。数值及完整效果唯一维护在提案§5；这里维护身份映射。分组与排序同提案§5（基本功13、江湖武学21、名家武学27、宗师武学10、绝学13）；标注（本轮新增）的12条为本轮由2招补为3招时占用的m03编号。删除或改名按§2墓碑规则处理，不得复用编号。

### 基本武功（13门各1招，共13）

| 稳定ID | 招式 |
|---|---|
| `basic_unarmed/m01` | 冲拳 |
| `basic_parry/m01` | 架挡 |
| `basic_neigong/m01` | 护息 |
| `basic_qinggong/m01` | 挪步 |
| `basic_blade/m01` | 劈斩 |
| `basic_sword/m01` | 点刺 |
| `basic_literacy/m01` | 察势 |
| `basic_staff/m01` | 横拦 |
| `basic_hidden_weapon/m01` | 连珠投 |
| `basic_spear/m01` | 突刺 |
| `basic_whip/m01` | 缠足 |
| `basic_mace/m01` | 重敲 |
| `basic_hook/m01` | 扯刃 |

### 江湖武学（7门各3招，共21）

| 稳定ID | 招式 |
|---|---|
| `thunder_sword/m01` | 疾电刺 |
| `thunder_sword/m02` | 雷回夺势 |
| `thunder_sword/m03` | 惊雷趁隙（本轮新增） |
| `bloody_blade/m01` | 血战冲阵 |
| `bloody_blade/m02` | 浴血斩 |
| `bloody_blade/m03` | 破阵连刀 |
| `hunyuan_neigong/m01` | 混元护息 |
| `hunyuan_neigong/m02` | 聚气归元 |
| `hunyuan_neigong/m03` | 清浊守一（本轮新增） |
| `talang_jue/m01` | 浪上轻身 |
| `talang_jue/m02` | 逆浪回步 |
| `talang_jue/m03` | 踏浪穿隙（本轮新增） |
| `fuhu_jin/m01` | 崩山伏虎 |
| `fuhu_jin/m02` | 拿虎卸劲 |
| `fuhu_jin/m03` | 镇步伏形（本轮新增） |
| `tiebi_gong/m01` | 铜墙架 |
| `tiebi_gong/m02` | 铁壁迎击 |
| `tiebi_gong/m03` | 守中蓄劲（本轮新增） |
| `wuzhen_pian/m01` | 知隙 |
| `wuzhen_pian/m02` | 澄心明鉴 |
| `wuzhen_pian/m03` | 照见虚实（本轮新增） |

### 名家武学（9门各3招，共27）

| 稳定ID | 招式 |
|---|---|
| `xiulou_daofa/m01` | 斩魂破阵 |
| `xiulou_daofa/m02` | 修罗追命 |
| `xiulou_daofa/m03` | 血刃回身 |
| `taiyi_jianfa/m01` | 太乙分光 |
| `taiyi_jianfa/m02` | 以静制动 |
| `taiyi_jianfa/m03` | 守一归锋（本轮新增） |
| `dongting_xinf/m01` | 水纳百川 |
| `dongting_xinf/m02` | 定波守心 |
| `dongting_xinf/m03` | 镜湖倒影（本轮新增） |
| `lingxu_bu/m01` | 凌虚纵影 |
| `lingxu_bu/m02` | 步罡抢势 |
| `lingxu_bu/m03` | 虚实换步 |
| `bawang_qiang/m01` | 一线贯甲 |
| `bawang_qiang/m02` | 回马枪 |
| `bawang_qiang/m03` | 压阵封门（本轮新增） |
| `youlong_bian/m01` | 双游连环 |
| `youlong_bian/m02` | 缚龙引 |
| `youlong_bian/m03` | 回风牵影 |
| `lieshi_jian/m01` | 碎甲重击 |
| `lieshi_jian/m02` | 叠嶂反震 |
| `lieshi_jian/m03` | 沉锏定势（本轮新增） |
| `suolang_gou/m01` | 拆兵钩 |
| `suolang_gou/m02` | 锁浪借势 |
| `suolang_gou/m03` | 双钩断流 |
| `fengmo_gun/m01` | 乱披风 |
| `fengmo_gun/m02` | 盘龙护身 |
| `fengmo_gun/m03` | 疯魔夺势 |

### 宗师武学（4门：3门各3招＋金刚不坏1招，共10）

| 稳定ID | 招式 |
|---|---|
| `poxu_zhi/m01` | 透隙一指 |
| `poxu_zhi/m02` | 截脉封劲 |
| `poxu_zhi/m03` | 凝虚破障（本轮新增） |
| `mantian_huayu/m01` | 花雨散星 |
| `mantian_huayu/m02` | 雨落无声 |
| `mantian_huayu/m03` | 天罗花幕 |
| `taxue_wuhen/m01` | 雪上无痕 |
| `taxue_wuhen/m02` | 不留影壁 |
| `taxue_wuhen/m03` | 踏雪折影（本轮新增） |
| `jingang_buhuai/m01` | 金刚护体 |

### 绝学（5门：三门各3招＋长生诀、慈航剑典各2招，共13）

| 稳定ID | 招式 |
|---|---|
| `csj_changshengjue/m01` | 生息归元 |
| `csj_changshengjue/m02` | 阴阳相济 |
| `jyg_zhanshentu_dao/m01` | 战神凝势 |
| `jyg_zhanshentu_dao/m02` | 神意护体 |
| `jyg_zhanshentu_dao/m03` | 天地合势 |
| `tiandao_blade/m01` | 问刀藏锋 |
| `tiandao_blade/m02` | 归鞘收势 |
| `tiandao_blade/m03` | 天刀一线 |
| `cihang_jiandian/m01` | 剑心通明 |
| `cihang_jiandian/m02` | 慈航照影 |
| `tianmo_ce/m01` | 魔影侵心 |
| `tianmo_ce/m02` | 天魔幻身 |
| `tianmo_ce/m03` | 魔极蚀心 |
