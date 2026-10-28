# 武功主动招式 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按用户确认后的两份设计文档实施84个武功专属主动招式：即时结算与普攻并行、角色出招CD与逐招技能CD双冷却、统一动作结算引擎、三NPC试点公开意图、网页招式栏与合并进度条。数值为候选，实施须先重跑预算并产出平衡报告。**本轮待用户确认，未开工。**

**Architecture:** model层定义招式契约与时钟纯函数；gamedata加载校验resources/skills.json的moves子数组；manager层以现有mu保护新增施放入口、效果引擎、战斗绑定与NPC意图四个文件，并把三条战斗路径收敛到统一动作结算；storage可选字段兼容旧档；app层注册两条指令并单增100ms意图调度；前端新增composable与面板，施放走立即通道不进排队队列。

**Tech Stack:** Go、JSON、Vue3、Python平衡脚本、独立端口协议冒烟与真实浏览器验收。

**规格来源（冲突时的优先序）：**
- 数值、目录、验收门槛：[主动招式完整提案](../../design/active_martial_moves_proposal.md)（§4数值口径、§5八十四招目录、§7NPC试点、§10平衡门槛）
- 数据契约、触发域、协议、存档、前端：[主动招式实现规范](../../design/active_martial_moves_implementation.md)（§2数据契约与Op白名单、§3战斗身份、§4双CD时钟、§5原子入口、§6统一结算顺序、§9前端接入）
- 稳定ID唯一来源：实现规范附录（84条冻结映射，含本轮新增12招的m03编号）
- 实施中确需改数值时：先修改提案明示数值并回写本计划，不得在代码或资源里暗改

## Global Constraints

- 不修改38个武功ID、物品/地图/NPC实体集合、MartialVersion=2、原入门与传承条件；不新增秘籍卷、技能点、地图或NPC；本轮无新增武功。
- 普攻2秒tick、用药10秒恢复冷却、烟尘粉撤离、快速完成规则一律不变；出招CD不是Player.State忙碌状态，不得阻断普攻/用药/撤离判断。
- 施放指令不进入useCommandQueue等待队列；客户端不上传伤害、等级、费用、攻速、CD或目标ID；所有deadline与随机判定服务端权威。
- 不读写正式存档：测试用t.TempDir、独立端口、独立MUD_DATA_DIR；随机判定注入固定种子/固定时钟，不真实等待。
- 候选数值未按"即时主动+普攻并行"新模型重跑预算前，不得标记平衡完成；发布阻断条件见提案§10。
- 先BuildOnly、StageOnly，部署单独等用户指令；本计划完成前不同步线上玩家手册的"已上线"表述。

### Task 1: 公式同步、模型契约与时钟纯函数

Files: docs/design/formulas.md；新增MudGame-server/internal/model/combat_moves.go、MudGame-server/internal/model/combat_move_clock.go；改MudGame-server/internal/model/types.go、internal/model/attributes.go。

Interfaces: `CombatMoveDef`/`MoveEffect`/`MoveCondition`/`CombatMoveState`/`CooldownWindow`/`NPCCombatArt`（实现规范§2）；`ActionDurationMs(baseMs int, effectiveSpeed float64) int64`；`MergeMoveBlock(old CooldownWindow, action, skill CooldownWindow, now int64) CooldownWindow`。

- [ ] 将获确认公式写入formulas.md：即时施放、双CD定义与合并窗口、攻速公式、S(L)=0.8+0.4L/(L+100)、D与倍率、效果结算顺序、切磋MaxHP/5预算、触发域；标注为实施基线而非已完成平衡。
- [ ] SkillDef新增`Moves []CombatMoveDef`；Player新增`CombatMoves CombatMoveState`；NPC新增`CombatArts []NPCCombatArt`；AttackSpeed保留int字段，运行时有效攻速用float64（不得把4.8截成4）。
- [ ] 测试：攻速4/4.8/5/8对应4000/3334/3200/2000ms；极大攻速下限1000ms；非正/NaN/Inf按基准4；技能CD不受攻速影响。MergeMoveBlock覆盖规范§4三行表（A剩12秒不回满、剩2秒延至4秒且保留起点、已可用新起满条）；外部clock由manager注入，测试不真实等待。

### Task 2: 资源加载、配置校验与84招写入

Files: 改MudGame-server/internal/gamedata/static_loader.go；新增MudGame-server/internal/gamedata/combat_move_validation.go；改MudGame-server/resources/skills.json（全部moves，ID按实现规范附录）；改internal/storage/store.go、internal/manager/runtime_manager_persistence.go。

Interfaces: `ValidateCombatMoves(skills map[string]*model.SkillDef) error`（容器类型按现状调整，不得建第二份技能表）。

- [ ] 校验：moveID归属且唯一、每门1～3招、cooldownSec 10～60、actionMs=4000、MPCost>0、浮点有限、倍率/比例/段数界限、Op/Trigger/Scope组合白名单、Conditions合法枚举；拒绝未知字段与未知枚举，无运行时eval。
- [ ] 加载器显式逐字段映射moves；diskPlayer、LoadPlayers、SavePlayers、clonePlayersMap补`combatMoves`可选字段，Skills/Blocks深拷贝，旧档缺字段读零值。
- [ ] 无moves的旧资源整包可启动；带moves的发布包必须覆盖全部现有武功，不接受半份目录；发布目录测试固定38/84，加载器不硬编码武功总数。
- [ ] 按提案§5与附录ID逐招写入resources（惊雷趁隙`durationMs:1000`是条件阈值、守中蓄劲带实付费用账本语义等特殊字段语义按§2）；ParryReduction存削减幅度（"乘0.5"存0.5）。

### Task 3: 战斗绑定与统一动作结算

Files: 新增MudGame-server/internal/manager/runtime_manager_combat_sessions.go、runtime_manager_combat_action.go、runtime_manager_combat_effects.go；改internal/manager/runtime_manager_combat.go、runtime_manager_player.go、runtime_manager_dungeon.go、runtime_manager.go。

Interfaces: `ActorKey`/`BattleBinding`/`RecentCombatEvent`（实现规范§3）；`resolveCombatActionNoLock(action CombatAction) CombatActionOutcome`；`beginCombatBindingNoLock`/`endCombatBindingNoLock`。

- [ ] Kill/Spar成功进入交战即绑定服务端会话与不可猜BattleID；换目标重建绑定并清旧目标效果/最近事件/待回疗，保留角色CD；撤离、死亡、认负、烟尘粉、实例销毁、`resolveOtherCombatantsTargetingNoLock`转目标统一走endCombatBinding。
- [ ] 三条战斗路径（PvP含离线、公共NPC、副本NPC）改实体适配器并共用结算：先手逐次计算逐次应用（先手减益影响本轮后手，废弃双方预计算）；整动作先算整数预算再按段序拆分（总5拆3段=2/2/1），各段独立命中/波动/暴击/招架，命中15%～95%、暴击≤25%、本招命中加成≤25个百分点。
- [ ] 防护同组取强；先扣护势穿透（≤30%），主动最终减伤乘区≤65%；技能防御削弱与忽略合计≤基准防御50%；次数效果最长12秒，先手随后两个全局轮。切磋为整动作设floor(MaxHP/5)预算，每段/每次HP变化后检查1HP、20%认负、NPC25%停止线；自损先验支付（不致死不触线）否则整招拒绝。
- [ ] 反击每来袭至多一次、取预算最大者、procDepth=1：不触发反击/吸血/MP返还/镜湖回疗，使用触发时有效属性与原姿态等级；dot不消耗来袭护体、不触发反应、不启动任何CD；施放当次不倒算次数。

### Task 4: 施放入口、双CD与协议

Files: 新增MudGame-server/internal/manager/runtime_manager_combat_moves.go、MudGame-server/internal/engine/workflow_combat_moves.go、MudGame-server/internal/app/handlers_combat_moves.go；改engine RuntimeRepository（GetCombatMoves/CastCombatMove/ProcessCombatMovesTick）、internal/app/command_registry.go、internal/app/server.go。

Interfaces: `CombatMovesView`/`CombatMoveResult`/`CombatMovesUpdate`（实现规范§5字段清单）；协议`get_combat_moves`、`cast_combat_move{moveId,battleId,expectedSequence}`。

- [ ] 锁内原子流程：重核身份/目标/绑定/安全区→expectedSequence校验→只读计划（纯治疗/护体收益预检）→扣费与自损支付→立即结算→攻速快照记Action与Skills→更新本人全部已有招式Blocks（≤84项，连续阻塞保持StartedAt）→sequence+1→构建回包后锁外发送；失败不扣费、不进CD、不耗随机数、不推进序号。
- [ ] 相同expectedSequence重放返回已处理结果与最新view，不再执行；并发同seq仅一次成功；角色A的出招CD不阻塞角色B；持锁不等待4秒、不做网络I/O。
- [ ] get_combat_moves：capabilities含combat_moves_v1、serverEpoch（进程随机串）、revision单调计数、serverTime、battleId、sequence、mp、effectiveActionSpeed、action窗口、逐招skillStartedAt/skillReadyAt/blockedSince/blockedUntil/unavailableReason；非战斗可查已装备招式，battleId为空。公共推送仅披露可观察信息与NPC意图，不下发对手技能CD/MP/sequence。
- [ ] 持久化与重启：CD存Unix毫秒真实时间终点，重启保留；旧BattleID、临时效果、NPC意图作废，TargetID/CombatType清理，COMBAT归IDLE（DEAD保持）；不执行切磋结束补满血路径；加载校验序号/窗口/毫秒范围并清理过期项；改名经别名映射迁移，不映射按钮位置；发布记录声明回滚丢字段的兼容限制。

### Task 5: 84招表驱动测试与12新招专测

Files: MudGame-server/internal/manager/新增招式测试文件（表驱动，随机注入）。

- [ ] 每招至少断言：正常施放的实际HP/效果次数、缺MP拒绝、技能CD拒绝、出招CD拒绝、未装备/兵器不符/非法目标拒绝；不只检查日志文本。
- [ ] 12新招专测：惊雷趁隙按目标actionReadyAt-now≥1000判定；踏浪穿隙recent_dodge窗口3999ms通过/4000ms拒绝、异目标拒绝；守中蓄劲返还包括折扣后的实付上限（floor(4S)/次、min(floor(8S),floor(实付/2))）；镜湖倒影只认move来袭、下一tick前认负取消；沉锏定势只匹配≥2段主动且落空也消耗；凝虚破障先自然扣次再额外扣次；踏雪折影多段合一不累计旧伤；镇步伏形不移装备不倒改CD。
- [ ] 触发域回归：察势/知隙/战神凝势被普攻、反击、dot均不消耗，下次主动招式消耗一次；来袭计数普攻/主动/反击命中或落空都消耗、多段整体一次、dot与自损不消耗；聚气归元折扣仅其他主动成功扣费时消耗。

### Task 6: NPC试点与100ms意图调度

Files: 新增MudGame-server/internal/manager/runtime_manager_combat_ai.go；改internal/app/server.go（单增100ms ticker，仅在意图创建/取消/结算时推送）；改MudGame-server/resources/npcs.json（三试点combatArts）。

- [ ] 三试点按提案§7：宋缺td_song_que（MP150，气血<40%优先归鞘收势，否则问刀藏锋）、庞斑残影lj_pang_ban（MP120，气血<50%优先天魔幻身，否则魔影侵心）、张绣db_zhang_xiu（MP60，就绪且MP足则一线贯甲）；level=100只算招式数值，不叠加武功被动；教学Skills表与原破招路径（张绣火烧浮桥等）不变。
- [ ] 开战AutoRounds≥1先给普攻tick；两CD就绪且MP足时发布intent{moveId,name,targetId,announcedAt,releaseAt=now+2000}；到公开时刻调同一施放引擎即时结算并开双CD；预告与出招CD期间普攻继续；100ms精度不保证零误差。
- [ ] 意图公开后不换招不换目标；目标失效、认负/死亡、MP不足、条件失效取消且不扣费不启动CD；不读玩家未公开请求、不预抽伤害随机数；围攻确定性选当前目标，实体共享MP/CD，同模板不同实例互不共享；MP耗尽只普攻。
- [ ] 断线不取消合法交战，退出副本实例取消意图；重启作废未执行预告并清空意图。

### Task 7: 前端招式栏与立即发送通道

Files: 新增MudGame/src/composables/useCombatMoves.js、MudGame/src/game/combatMoveClock.js、MudGame/src/components/GameCombatMovesPanel.vue；改MudGame/src/components/GameActionPanel.vue、MudGame/src/views/GameView.vue、MudGame/src/composables/useCommandQueue.js。

Interfaces: `useCombatMoves({ sendImmediateCommand, addLog, playerId, monotonicNow })`暴露view/pending/applySnapshot/castMove/reset/dispose；`cooldownFraction(window, now)`纯函数。

- [ ] useCommandQueue新增sendImmediateCommand：与原command共享15条/2秒额度，用尽立即返回false并提示，绝不进入等待队列、不设flush timer；发送失败解除pending，不扣MP不启动CD；禁止并发多重pending；断线/换角色清pending与视图。
- [ ] (serverEpoch,revision)忽略旧包；跨epoch仅在已确认新连接上reset；serverTime校时`displayNow=serverTime+(performance.now()-receivedMono)`，revision不旧才修正；浏览器不自行授权施放、不预测伤害、不预扣费用。
- [ ] 面板按五位置分组纯文字按钮，展示实际可用的最多15招；20%透明度遮罩用rAF或≤100ms tick连续动画，≤10秒显示0.1秒向上取整、>10秒整数、未结束不显示0；展开详情分列"出招还需/本招还需"与基础4秒/实际出招时长；缺MP、兵器不符、条件不满足单独提示，不用冷却遮罩掩盖。
- [ ] 服务端未声明combat_moves_v1时隐藏可操作栏，不为旧服生成假按钮；旧客户端不请求技能照常普攻；技能可用性由服务端判定，前端倒计时只做显示。

### Task 8: 平衡验证、玩家文档与交付

Files: 新增scripts平衡脚本与docs/design/active_martial_moves_balance_results.md；改scripts/generate_player_guide.py及玩家文档；全量门禁。

- [ ] 同经验/潜能/装备预算的9类兵器配置，L=20/100/200，固定100种子、单局上限120秒，三策略（仅普攻、按可用按钮顺序、按受击/护体/弱点选择）；记录胜率、P50/P90时长、普攻时间戳与原tick一致性、主动伤害/治疗/减伤占比、MP净耗、各招使用次数、两类CD各自受阻次数、卡死局；报告存balance_results并列种子与配置。
- [ ] 三试点用其章节之前可正常获取的武功/装备验证，不用首领掉落物作前置；发布阻断核对：无预告开场特殊伤害、基准配置胜率下降超20个百分点或低于50%、无响应窗口内主动组合普遍击倒同预算满血目标、任何超界/循环/永久僵局；失败调整提案明示数值后重测，不暗加隐藏倍率。
- [ ] 玩家文档新增「武功与招式」页：38门完整目录、装备/兵器条件、基础与实际出招时长、每招技能CD、费用、效果与触发域说明；生成器读取真实moves输出，不在确认前写成已上线。
- [ ] 提案与实现规范回写实施结果、数值定稿与验收证据，formulas.md与最终数值一致性核对；中文提交；Go build/vet/test、前端test/lint/build、隔离协议冒烟、真实WS与浏览器验收通过后BuildOnly→StageOnly，部署等用户指令。
