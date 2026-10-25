# 玩家与 NPC 属性系统设计

> **状态（2026-10）：本设计的迁移阶段已全部实施完毕**——分层属性结构、`EffectiveAttributes` 计算链路（`effectiveAttributesForPlayerNoLock`）、战斗公式接入、前端状态面板均已落地。本文保留作为**分层思想、字段命名与境界设计的背景依据**；“当前属性系统评估”一节描述的是重构前的旧状态，仅作历史参考。
>
> **现行数值公式（衍生公式、气血/内力上限、虚弱、加成表）以 [formulas.md](formulas.md) 为唯一权威**，本文的公式段落如与之冲突，以 formulas.md 为准。

本文档用于在进入任务、对话、势力和更复杂战斗系统前，重新规划玩家与 NPC 的属性体系。

目标不是完全推翻当前实现，而是在现有 `Attributes` 基础上分层扩展，避免后续每做一个玩法就零散添加字段。

## 当前属性系统评估（重构前旧状态，历史参考）

当前后端 `model.Attributes` 包含以下类型的字段：

- 成长资源：`experience`、`potential`
- 生存资源：`currentHp`、`maxHp`、`currentMp`、`trainedMaxMp`、`maxMp`
- 先天属性：`strength`、`constitution`、`agility`、`intelligence`、`appearance`、`luck`
- 战斗派生：`attack`、`defense`、`hitRate`、`dodge`、`parry`、`critRate`、`attackSpeed`
- 后天属性：`postStrength`、`postConstitution`、`postAgility`、`postIntelligence`

### 优点

- 足够支撑当前版本的基础战斗、疗伤、打坐、学习和副本。
- 玩家和 NPC 共用同一结构，副本 NPC 实例化成本低。
- 字段直观，前端展示和 JSON 配置都比较简单。
- 已经有“先天属性 + 后天属性”的雏形，适合武侠成长体系。

### 主要问题

- 属性没有分层，成长资源、当前状态、基础属性、派生战斗值混在一起。
- `attack`、`defense`、`hitRate` 等派生值目前像静态字段，后续装备、技能、状态、境界介入后会难以维护。
- NPC 与玩家使用同一结构是好事，但缺少 NPC 专用倾向，例如境界、气势、警觉、战斗风格、掉落权重。
- 社交、任务、势力、对话判断缺少属性支撑，目前只有 `appearance`、`luck` 可用。
- 黄易式武侠需要表现心性、悟性、根骨、气势、精神压迫、宗师境界等内容，现有字段承载不足。
- 如果继续直接在 `Attributes` 里平铺字段，以后会越来越难知道哪些是原始值，哪些是计算值。

## 设计原则

### 1. 分层而不是堆字段

属性应分为：

- 基础属性：角色天生或长期成长形成的底层能力。
- 资源属性：当前气血、内力、精神等会变化的数值。
- 派生属性：由基础属性、技能、装备、状态计算得出。
- 江湖属性：影响对话、任务、声望、交易、奇遇。
- 抗性与状态：影响中毒、封穴、震慑、魅惑等效果。
- 成长进度：经验、潜能、境界、修行阶段。

### 2. 玩家和 NPC 使用同一核心结构

玩家、静态 NPC、副本 NPC 应共享同一套核心属性结构。NPC 可以少填字段，后端用默认值补齐。

这样后续统一战斗实体时，玩家和 NPC 都能进入同一套公式。

### 3. 静态配置填基础值，运行时计算派生值

资源 JSON 里应该尽量填写基础值、境界、战斗风格和少量修正，不要手写完整派生值。

例如 NPC 配置不应长期手填所有 `attack/hitRate/dodge/parry`，而应填写：

- `base.strength`
- `base.agility`
- `skills`
- `realm`
- `combatStyle`
- 可选 `derivedOverrides`

派生值由后端统一计算。

### 4. 兼容当前字段，但逐步迁移

短期可以保留当前 `Attributes` 字段，避免大拆导致功能回退。

中期新增结构化字段：

- `base`
- `resources`
- `derived`
- `social`
- `resistances`
- `cultivation`

长期再逐步移除或弱化旧的平铺字段。

## 推荐属性结构

建议未来的角色属性采用以下结构。

```json
{
  "level": 1,
  "realm": "ordinary",
  "experience": 0,
  "potential": 500,
  "base": {
    "strength": 20,
    "constitution": 20,
    "agility": 20,
    "perception": 20,
    "willpower": 20,
    "charm": 20,
    "luck": 20
  },
  "growth": {
    "postStrength": 0,
    "postConstitution": 0,
    "postAgility": 0,
    "postPerception": 0,
    "postWillpower": 0
  },
  "resources": {
    "hp": 100,
    "maxHp": 100,
    "mp": 80,
    "trainedMaxMp": 80,
    "maxMp": 200,
    "spirit": 100,
    "maxSpirit": 100
  },
  "derived": {
    "attack": 15,
    "defense": 15,
    "hitRate": 100,
    "dodge": 100,
    "parry": 100,
    "critRate": 1,
    "attackSpeed": 4,
    "initiative": 100
  },
  "social": {
    "fame": 0,
    "notoriety": 0,
    "authority": 0,
    "credit": 0
  },
  "resistances": {
    "poison": 0,
    "bleed": 0,
    "stun": 0,
    "fear": 0,
    "charm": 0,
    "seal": 0
  }
}
```

## 属性分层说明

### 成长属性

| 字段 | 含义 | 适用对象 |
| --- | --- | --- |
| `level` | 简化等级，可用于内容门槛和副本缩放 | 玩家、NPC |
| `realm` | 武道境界，例如凡人、入门、好手、一流、宗师 | 玩家、NPC |
| `experience` | 经验，代表历练积累 | 玩家 |
| `potential` | 潜能，学习技能消耗 | 玩家 |

建议：

- 玩家可以继续使用经验和潜能。
- NPC 不一定需要经验和潜能，但可以有 `level` 和 `realm`。
- 高级原著人物应主要靠 `realm`、技能和特殊标签表现，不靠堆超大数值。

### 基础属性

建议把现有四维扩展为五到七维。

| 新字段 | 当前对应 | 含义 |
| --- | --- | --- |
| `strength` | `strength` | 臂力、爆发、兵器伤害 |
| `constitution` | `constitution` | 根骨、气血、抗打击 |
| `agility` | `agility` | 身法、闪避、出手顺序 |
| `perception` | `intelligence` 的一部分 | 悟性、识破、学习效率、机关古卷 |
| `willpower` | 新增 | 心志、抗压、内功稳定、抵御魅惑/恐惧 |
| `charm` | `appearance` 调整 | 容貌、气质、交涉、NPC 初始态度 |
| `luck` | `luck` | 奇遇、掉落、极低概率事件 |

说明：

- 当前 `intelligence` 这个词偏现代和宽泛，建议改为 `perception` 或 `comprehension`。
- 黄易武侠里“悟性”和“心志”非常重要，建议拆成 `perception` 与 `willpower`。
- `appearance` 可以保留显示名为“容貌”，内部改为 `charm` 更适合社交系统。

### 后天成长

当前已有：

- `postStrength`
- `postConstitution`
- `postAgility`
- `postIntelligence`

建议调整为：

- `postStrength`
- `postConstitution`
- `postAgility`
- `postPerception`
- `postWillpower`

含义：

- 基础属性是创建角色时确定或少量改变。
- 后天属性来自技能、修行、装备、称号、境界。
- 总属性 = 基础属性 + 后天属性 + 临时状态修正。

### 资源属性

保留现有：

- `currentHp`
- `maxHp`
- `currentMp`
- `trainedMaxMp`
- `maxMp`

建议新增：

- `spirit`
- `maxSpirit`

用途：

- 对话、心魔、魅惑、恐惧、宗师威压、秘境幻境。
- 避免所有非肉体伤害都挤进 HP 或 MP。

命名建议：

- 前端显示：气血、内力、心神。
- 后端字段：`hp`、`mp`、`spirit`。

短期可以继续保留 `currentHp/currentMp`，中期再迁移。

### 派生战斗属性

当前字段可以保留：

- `attack`
- `defense`
- `hitRate`
- `dodge`
- `parry`
- `critRate`
- `attackSpeed`

建议新增：

- `initiative`：先手、反应、出手时机。
- `guardBreak`：破招、破防。
- `damageReduction`：减伤。

但这些字段不建议长期由 JSON 手填，而应由计算得出：

```text
derived = base + growth + skills + equipment + realm + buffs + overrides
```

NPC 可以允许 `derivedOverrides`，用于快速调怪。

### 社交与江湖属性

建议新增 `social`，用于任务、对话和势力。

| 字段 | 含义 |
| --- | --- |
| `fame` | 江湖名声，正面知名度 |
| `notoriety` | 恶名、通缉、黑道关注 |
| `authority` | 官府/军府威望 |
| `credit` | 商会信用、交易可信度 |

说明：

- 势力声望仍应放在 `reputations map[string]int`。
- `social` 是角色整体江湖形象。
- 对话系统可以同时判断 `charm`、`fame`、`notoriety`、势力声望。

### 抗性与状态

建议新增 `resistances`：

- `poison`：毒抗
- `bleed`：流血抗性
- `stun`：眩晕抗性
- `fear`：恐惧/威压抗性
- `charm`：魅惑抗性
- `seal`：封穴/封内抗性

后续状态系统可以新增：

```json
{
  "effects": [
    {
      "id": "poison_minor",
      "duration": 3,
      "stacks": 1
    }
  ]
}
```

不建议把所有状态都做成 `Attributes` 的字段，否则以后会失控。

## 境界设计

黄易式武侠非常适合做境界系统。建议使用 `realm` 作为玩家和 NPC 的高层定位。

第一版境界：

| ID | 显示名 | 用途 |
| --- | --- | --- |
| `ordinary` | 凡人 | 普通百姓、商贩、低级玩家 |
| `trainee` | 初学 | 初入武馆、基础训练 |
| `fighter` | 好手 | 普通江湖人、镖师、府卫 |
| `elite` | 一流 | 门阀客卿、精锐刺客、高手 NPC |
| `master` | 宗师 | 高级原著人物、势力代表 |
| `grandmaster` | 大宗师 | 石之轩、宋缺等顶级人物 |
| `transcendent` | 破碎边缘 | 传鹰、庞斑、浪翻云等传说层 |

用途：

- 控制 NPC 难度。
- 控制任务和对话压迫感。
- 控制战斗公式修正。
- 控制玩家是否能识破、挑战、学习。

建议：

- 早期不开放玩家快速升境界。
- 原著人物的强大主要由境界、技能和特殊机制体现。
- 不要只靠 HP/Attack 堆数值，否则高阶人物会显得像普通怪物加强版。

## 玩家属性建议

玩家创建时仍可分配 80 点基础属性，但建议从四维改为五维：

- 臂力 `strength`
- 根骨 `constitution`
- 身法 `agility`
- 悟性 `perception`
- 心志 `willpower`

容貌与福缘建议不再参与 80 点分配：

- 容貌 `charm`：随机或少量可选。
- 福缘 `luck`：随机或账号/角色创建时生成，不轻易改变。

原因：

- 容貌和福缘如果可自由分配，容易变成最优解或废属性。
- 悟性与心志对黄易世界更重要，应进入核心分配。

推荐初始：

```json
{
  "strength": 16,
  "constitution": 16,
  "agility": 16,
  "perception": 16,
  "willpower": 16,
  "charm": 20,
  "luck": 20
}
```

## NPC 属性建议

NPC 配置应允许简写，减少内容制作负担。

普通 NPC：

```json
{
  "level": 1,
  "realm": "ordinary",
  "base": {
    "strength": 8,
    "constitution": 10,
    "agility": 10,
    "perception": 12,
    "willpower": 10,
    "charm": 12,
    "luck": 10
  }
}
```

战斗 NPC：

```json
{
  "level": 8,
  "realm": "fighter",
  "combatStyle": "blade",
  "base": {
    "strength": 18,
    "constitution": 14,
    "agility": 16,
    "perception": 10,
    "willpower": 12
  },
  "skills": {
    "basic_blade": 80,
    "basic_parry": 60,
    "basic_qinggong": 40
  }
}
```

原著高手：

```json
{
  "level": 80,
  "realm": "grandmaster",
  "combatStyle": "special",
  "base": {
    "strength": 45,
    "constitution": 42,
    "agility": 48,
    "perception": 55,
    "willpower": 60,
    "charm": 45,
    "luck": 35
  },
  "specialTraits": ["宗师威压", "精神破绽", "天人感应"]
}
```

## 属性计算建议

### 第一版动态战斗公式

玩家的战斗属性不再使用创建角色时写死的数值。计算顺序为：

1. 合并先天属性。
2. 合并技能、成长带来的后天属性。
3. 合并装备提供的臂力、根骨、身法、悟性和心志。
4. 使用以下公式计算基础战斗属性。
5. 最后叠加技能和装备直接提供的攻击、命中、防御、招架、躲闪加成。

```text
攻击 = 10 + 臂力 × 2 + 身法 × 0.5
命中 = 50 + 悟性 × 2 + 身法
防御 = 5 + 根骨 × 2 + 臂力 × 0.5 + 心志 × 0.5
招架 = 30 + 臂力 + 悟性 + 身法 × 0.5
躲闪 = 30 + 身法 × 2 + 悟性 × 0.5
```

所有除法使用向下取整。示例：臂力16、根骨16、身法16、悟性16、心志16时：

```text
攻击 = 50
命中 = 98
防御 = 53
招架 = 70
躲闪 = 70
```

该设计使属性关系保持清晰：

- 臂力主要提高攻击，同时影响防御和招架。
- 根骨主要提高防御。
- 身法主要提高躲闪，同时影响攻击、命中和招架。
- 悟性主要提高命中和招架，同时少量提高躲闪。
- 心志参与防御，后续还可接入精神抵抗。
- 技能和装备既可以增加基础属性，也可以在公式完成后提供直接加成。

建议后端增加统一计算函数：

```go
func BuildEffectiveAttributes(entity Entity) EffectiveAttributes
```

计算顺序：

1. 读取基础属性。
2. 加上后天成长。
3. 加上技能加成。
4. 加上装备加成。
5. 加上境界修正。
6. 加上状态效果。
7. 应用 NPC 或副本临时修正。

最终战斗系统只使用 `EffectiveAttributes`。

好处：

- 装备、技能、状态、境界都能自然接入。
- 前端显示可以区分“基础值”和“有效值”。
- NPC 配置更轻，不需要手填所有派生值。

## 与当前代码的迁移关系

### 第一阶段：不破坏现有结构

保留当前 `Attributes` 字段，新增结构化字段或临时转换函数。

建议先做：

- 新增文档和设计。
- 新增 `EffectiveAttributes` 计算结构。
- 当前 JSON 仍可继续填平铺字段。
- 后端在计算时把旧字段转换为新结构。

### 第二阶段：资源 JSON 改为新结构

逐步将 NPC 和玩家默认属性改为：

- `level`
- `realm`
- `base`
- `resources`
- `derivedOverrides`

静态 loader 负责兼容旧平铺字段一段时间。

### 第三阶段：前端显示重构

状态面板改成：

- 修为：经验、潜能、境界。
- 气脉：气血、内力、心神。
- 根基：臂力、根骨、身法、悟性、心志、容貌、福缘。
- 战斗：攻击、防御、命中、闪避、招架、暴击、先手。
- 江湖：名声、恶名、官威、信用。

### 第四阶段：战斗公式接入

战斗系统不再直接读 `attacker.Attributes.Attack`，而是读计算后的有效属性。

## 推荐字段命名

短期为了少改代码，可以继续保留：

- `currentHp`
- `maxHp`
- `currentMp`
- `trainedMaxMp`
- `maxMp`
- `strength`
- `constitution`
- `agility`
- `intelligence`
- `appearance`
- `luck`

中期建议新字段：

- `hp` 替代 `currentHp`
- `mp` 替代 `currentMp`
- `spirit` 新增
- `perception` 替代 `intelligence`
- `charm` 替代 `appearance`
- `willpower` 新增

对外显示：

- `strength`：臂力
- `constitution`：根骨
- `agility`：身法
- `perception`：悟性
- `willpower`：心志
- `charm`：容貌
- `luck`：福缘

## 当前最推荐的实现步骤

在第七阶段任务/对话系统之前，建议插入一个新阶段：

### 阶段 6.5：属性系统重构第一版

目标：

- 不大规模破坏现有玩法。
- 建立后续任务、对话、战斗、原著人物都能使用的属性基础。

任务：

1. 新增结构化属性模型，但保留旧字段。
2. 新增 `EffectiveAttributes` 计算函数。
3. 玩家创建时使用新五维基础属性概念。
4. NPC JSON 开始支持 `level`、`realm`、`base`、`resources`、`derivedOverrides`。
5. 战斗暂时仍可使用旧字段，但测试 `EffectiveAttributes` 计算。
6. 前端状态面板暂不大改，只保证新字段不会破坏旧展示。

验证：

- `go test ./...` 通过。
- `npm run build` 通过。
- 新角色创建正常。
- NPC 和副本 NPC 正常加载。
- 现有战斗不回退。

## 总结

当前属性系统可以在原基础上修改，不需要推倒重来。

最关键的调整是：

- 把属性分层。
- 区分基础值和派生值。
- 新增悟性、心志、心神、境界、社交、抗性。
- NPC 配置尽量简写，由后端统一计算派生值。

这样后续开发任务、对话、势力、秘境、宗师、战斗深化时，不需要每遇到一个新需求就临时添加零散字段。
