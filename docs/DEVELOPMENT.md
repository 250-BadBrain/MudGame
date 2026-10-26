# 统一开发指导文档

本文档是破碎江湖（黄易武侠 MUD）的**统一开发入口**：说明各仓库与模块的功能、如何使用，并规定标准的开发流程（先读什么 → 先改什么文档 → 改哪些代码 → 如何测试 → 文档如何留存）。任何新开发任务从本文档开始。

**游戏系统逐一说明（副本/战斗/学习/商店/状态机等每个系统的机制、代码位置、扩展方法与注意事项）见 [SYSTEMS.md](SYSTEMS.md)——接手或切换会话时，读完本文再读它即可上手。**

**想了解当前具体能玩什么、每条故事如何触发和走完，先看 [玩家手册](player/README.md)。** 它覆盖全部副本、城市交谈、任务与收藏，并区分实际接通和仅有配置的内容。修改资源后运行 `python scripts/generate_player_guide.py` 更新事实页，再用 `--check` 检查生成一致性与链接；新手推荐路线仍需人工同步，生成器不覆盖入口说明。

解锁、交谈、采集、物品使用等运行时规则变更时，还要人工核对并更新生成逻辑与手写说明；`--check` 验证生成内容与资源、链接的一致性，不代替实机游玩或行为测试。

## 1. 仓库结构与职责

**单一 Git 仓库（monorepo，240 轮迭代后）**，前后端与文档在同一仓库中一起提交、一起回退：

| 目录 | 职责 |
|---|---|
| `docs/` | 全部设计/内容文档（本文档所在地） |
| `MudGame/` | 前端（Vue 3 + Vite + Pinia + Vue Router），子目录内有自己的 `.gitignore` |
| `MudGame-server/` | 后端（Go，WebSocket 服务，默认端口 4009），子目录内有自己的 `.gitignore` |
| `start.bat` / `deploy.bat` | 本地启动 / 远程部署脚本 |
| `data/` | 后端运行时动态数据（玩家存档），**已在 .gitignore 忽略，禁止提交** |

> 历史备注：2026-10 由三个独立仓库（文档仓 + 前端仓 + 后端仓）通过 `git subtree` 合并为本单仓库，三段提交历史全部保留（导入提交 `2896cd5`/`00ebe20`）。注意 subtree 导入的历史中文件路径不带 `MudGame/`、`MudGame-server/` 前缀，因此 `git log -- MudGame/` 只能追溯到导入提交；更早的历史通过导入提交的第二个父提交浏览（`git log 2896cd5^2`）。

## 2. 运行与调试

```sh
# 后端（在 MudGame-server 目录，或直接用根目录 start.bat）
go run ./cmd/mudserver
# 环境变量：PORT（默认 4009）、MUD_DATA_DIR（默认 ../data/servers/server1）、MUD_RESOURCES_DIR（默认 resources/）

# 前端（在 MudGame 目录）
npm install
npm run dev        # vite 开发服务器，默认 5173，默认连接本机 4009
```

- 前端可用 `.env.local` 的 `VITE_WS_URL=ws://localhost:4009/ws` 指定后端。
- 生产部署（Cloudflare Pages + Tunnel）见 `MudGame/README.md`；远程部署用根目录 `deploy.bat`。
- **测试时不要污染正式数据目录**：用临时目录起独立服务，例如 `MUD_DATA_DIR=$(mktemp -d) PORT=14009 go run ./cmd/mudserver`。

### 后端部署（Windows → Linux）

根目录 `deploy.bat` 启动 PowerShell 7 的 `scripts/deploy.ps1`，默认连接 `oracle`，部署到 `/home/ubuntu/MudGame`，操作 systemd 的 `mud` 服务和 4009 端口。需要 PowerShell 7、Go、Git、OpenSSH、tar；远端需要 Python 3.10+、systemd，以及当前 SSH 用户可通过 `sudo -n` 停启该服务。

```powershell
.\deploy.bat -BuildOnly  # 本地测试、静态检查、交叉编译和打包，不连接服务器
.\deploy.bat -StageOnly  # 同上，并上传、校验资源和服务配置；不停服，清理远端校验包
.\deploy.bat             # 上传校验后，备份并切换；失败自动回滚程序和资源
```

- 每次从当前源码重新构建 Linux amd64 程序，不读取旧的 `MudGame-server/mud`；`GOOS` 等构建环境变量在结束时恢复。
- 本地发布包保存在 `artifacts/backend/<release-id>/`（Git 忽略），`release.json` 记录 Git revision、工作区是否有修改、构建时间及所有程序/资源文件的 SHA-256。发布包**只含程序、资源和清单，不含本地存档**。
- 上传完成、包校验和 systemd 配置预检通过后才停服。确认旧进程退出并落盘后，在 `.deploy/backups/<release-id>/` 保留原程序目录和当前存档快照，再切换新程序目录、启动并检查 `/health` 与实际运行的二进制。
- 正式存档始终保留原路径，默认 `data/servers/server1/`。服务自定义 `MUD_DATA_DIR` 时备份实际目录；存档不得放在待切换的 `MudGame-server/` 或 `.deploy/` 内。服务若自定义资源路径，必须仍指向该后端目录的 `resources/`。
- 切换失败时先停止新进程，再恢复旧程序目录、启动并检查健康状态。**回滚不会自动恢复存档快照**，避免覆盖期间产生的玩家进度；确需恢复时，由维护者停服后按备份目录的 `backup.json` 人工确认。
- 失败的新程序保存在 `.deploy/failed/<release-id>/`，备份不自动删除。备份失败同样会重新启动旧服务；如果回滚也失败，脚本以失败状态退出并输出备份位置。
- `.deploy/deploy.lock` 防止并发部署。SSH 异常中断可能留下锁；只有确认无部署进程运行后才可手动删除该锁。
- 可传 `-SshHost`、`-RemoteRoot`、`-ServiceName`、`-ExeName`、`-HealthPort`、`-OutputDirectory` 覆盖默认值。此脚本更新已有的运行中服务，不负责首次安装 systemd 单元。

部署脚本回归测试：`python -m unittest discover -s scripts/tests -v`。修改部署事务时，须覆盖成功切换、存档保留、包校验失败不停服和新服务失败后回滚；Linux 冒烟须用隔离网络或独立端口、临时存档，不能在正式服务上故意制造故障。

Linux 构建冒烟可运行 `python3 scripts/tests/smoke_backend.py --binary /path/to/mud --resources /path/to/resources`：自动选择独立端口和临时存档，通过真实 WebSocket 注册、登录、创角、选角、查询状态，停服后只为该测试角色写入通关旗标，再重启检查称号和存档重载。不要传入正式存档目录；该脚本不提供这种参数。

## 3. 后端各部分功能（MudGame-server）

请求链路：**WebSocket → app（网络层）→ engine（业务编排）→ manager（运行时状态）→ storage（落盘）**。静态资源由 gamedata 一次性加载。

| 目录/文件 | 功能 | 什么时候改这里 |
|---|---|---|
| `cmd/mudserver` | 启动入口 | 几乎不动 |
| `internal/app` | WS 接入、会话绑定、命令注册表（`command_registry.go`）、handler 分发、错误本地化（`error_localization.go`） | 新增指令类型、改会话/鉴权行为 |
| `internal/engine` | 对外服务接口与流程编排（`workflow.go` / `workflow_game.go` / `workflow_system.go`） | 新玩法的编排步骤、跨 manager 协调 |
| `internal/engine/runtime` | 副本实例管理（`dungeon_manager.go`）：模板加载、实例生命周期、机关状态机、目标进度、结算 | 副本机制（非内容）改动 |
| `internal/manager` | 玩家/房间/NPC/战斗/商店/学习等运行时内存状态与指令实现，按领域分文件（`_player` / `_combat` / `_dungeon` / `_room` / `_dialogue` / `_effective` / `_static_npc` / `_persistence`…） | 绝大多数玩法逻辑 |
| `internal/storage` | 动态数据与磁盘的读写（tmp+rename 原子写，30s 自动保存 + 退出 flush） | 新增持久化字段 |
| `internal/model` | 领域模型与**数值公式**（`attributes.go`、`skill_progression.go`、`types.go`） | 公式调整（先改 formulas.md！） |
| `internal/gamedata` | 静态资源加载与**一致性校验**（`static_loader.go`） | 新增资源字段、校验规则 |
| `internal/protocol` | 协议结构（CommandMessage / GameResult） | 协议字段变更 |
| `resources/` | 全部静态内容：`items.json`、`npcs.json`、`skills.json`、`factions.json`、`dialogues.json`、`dungeon_dialogues.json`、`quests.json`、`dungeons/*.json`（每副本一个文件）、`worlds/*.json`（洛阳等世界地图） | **内容开发的主战场** |

运行时约定（详见各设计文档）：

- 全内存运行，磁盘只在启动加载与定时/退出落盘时触碰；manager 内部用一把大锁 `r.mu` 保护，跨锁 I/O 要先收集后操作。
- 战斗 tick 2 秒、状态 tick 1 秒；死亡复活带 5 分钟虚弱（战斗属性减半）。
- 副本 NPC 一律 `instanceOnly: true`；叙事角色 `combatMode: "protected"`、氛围角色 `nonCombat: true`、剧情型 NPC 不可被击杀。
- 静态 NPC 商店库存是运行时隔离副本（`staticNPCStateNoLock`），不污染模板。

## 4. 前端各部分功能（MudGame/src）

| 目录/文件 | 功能 | 什么时候改这里 |
|---|---|---|
| `views/` | 页面级组件：登录/注册/创角/选角/`GameView.vue`（游戏主界面） | 页面级结构 |
| `components/` | 游戏界面子组件：房间面板、出口九宫格、信息面板（状态/技能/背包/装备/地图）、商店、动作面板、日志、聊天、势力、任务、侧栏工具条 | 具体某块 UI 的结构与样式 |
| `composables/` | 游戏逻辑复用单元：命令队列、日志、实体动作、副本视图、信息面板、商店学习、背包、键盘导航、聊天状态 | 交互逻辑 |
| `game/messageText.js` | 服务端消息/错误码 → 纯文本映射 | 新消息类型或错误码 |
| `protocal/message.js` | 客户端消息构造器（system / command / chat） | 协议变更 |
| `utils/ws.js` | WebSocket 连接状态机：心跳、断线重连、会话恢复、过期连接守卫 | 连接行为 |
| `stores/player.js` | Pinia 全局玩家态 | 全局共享状态 |
| `assets/styles/` | 全局样式 | 全局主题 |

**两条硬性架构约定**：

1. **composables 之间禁止互相 import**。每个 composable 只定义依赖（接收 ref/函数参数），由 `GameView.vue` 统一装配并注入。新增 composable 必须在 GameView 里完成组装。
2. **父组件 scoped CSS 只能命中子组件的根元素**。子组件内部样式写在子组件自己的 `<style scoped>` 里；GameView 里只保留布局层（面板容器、网格）样式。

## 5. 权威文档与“文档先行”规则

数值与内容以文档为唯一权威，**改任何数值前先改文档，再落实到代码**（formulas.md 文末有公式 → 代码位置对照表）：

| 文档 | 权威范围 |
|---|---|
| [SYSTEMS.md](SYSTEMS.md) | 游戏系统总览：每个系统的机制、代码位置、扩展方法与注意事项 |
| [design/formulas.md](design/formulas.md) | 全部数值公式：属性、战斗、恢复、经验曲线、潜能消耗、技能加成、虚弱 |
| [design/economy_design.md](design/economy_design.md) | 三资源（经验/潜能/银两）来源与消耗：采集/帮工收益、副本收益总表、装备掉落、商店定价、进阶武学链路 |
| [content/huangyi_dungeon_redesign.md](content/huangyi_dungeon_redesign.md) | 副本设计总纲：设计原则、章节链解锁、逐副本设计 |
| [player/README.md](player/README.md) | 当前玩家内容入口；七卷副本全攻略、城市交谈、任务与收藏，事实页由资源生成 |
| [content/dungeon_flows.md](content/dungeon_flows.md) | 前九关短版摘要，不作为全部副本清单 |
| [design/skill_learning_progression_design.md](design/skill_learning_progression_design.md) | 技能学习流程规则（老师上限、10 秒结算、停止条件） |
| [design/attribute_system_design.md](design/attribute_system_design.md) | 属性分层/字段命名/境界设计的背景依据（已实施） |
| [design/world_design_huangyi.md](design/world_design_huangyi.md) | 世界观与世界格局设计 |

文档索引统一登记在 [README.md](README.md)。新增文档按主题放入 `design/` 或 `content/` 子目录，不要堆在 `docs` 根目录。

## 6. 标准开发流程

### 第一步：读（按任务类型的必读清单）

| 任务类型 | 必读 |
|---|---|
| 新增副本章节 | 本文档 §3/§5 → dungeon 总纲（redesign）→ economy §1.3 收益表 → 玩家手册对应卷攻略 → 一个现成副本 JSON（复杂机关参考 `datang_yanggong.json`，多步机关参考 `posui_jingyan.json`） |
| 新物品/装备 | economy §2 定价定位 → `items.json` 现有条目 |
| 新技能/进阶武学 | formulas §3 加成表 → skill_learning 文档 → `skills.json` |
| 数值公式调整 | formulas 对应小节（含文末代码对照表定位代码） |
| 新指令/玩法 | `protocol/` → `app/command_registry.go` + 同类 handler → 对应 `engine/workflow*.go` → 对应 `manager/runtime_manager_*.go` → 前端 `messageText.js` + 相应 composable |
| 世界内容（房间/NPC/对话） | world_design_huangyi.md → `worlds/luoyang.json` → `npcs.json` / `dialogues.json` |
| UI 调整 | 目标 `components/*.vue` 与 `composables/*.js` → GameView 装配处 |

读代码时先看同领域 `*_test.go`：测试就是该模块行为的最新规格说明。

### 第二步：文档先行

动代码之前，把**设计决定写进权威文档**并让文档与将来的实现一致：

- 数值变动 → formulas.md（同步文末代码对照表）或 economy_design.md。
- 新副本 → 先在 dungeon 总纲和 economy 收益表明确设计；实现后生成玩家手册对应卷攻略，并人工核对新剧情的入口、操作与结局。`dungeon_flows.md` 仅保留前九关摘要。
- 新玩法且尚无文档 → 在 `docs/design/` 新建文档并在 README.md 登记，写清规则、数值、边界情况。

### 第三步：改（任务 → 文件对照）

| 任务 | 典型改动文件 |
|---|---|
| 新副本 | `resources/dungeons/<id>.json`（新文件）；`npcs.json` 加 instanceOnly NPC（ID 前缀按副本）；`items.json` 加掉落/残页；`dungeon_dialogues.json` 加剧情动作对话；（进阶武学）`skills.json` + 老师 NPC + 残页 learnSkill 效果 |
| 新物品 | `items.json`；来源：副本 `rewards.items` 或商店 NPC 配置 |
| 数值公式 | 按 formulas 文末对照表定位 `model/` 或 `manager/`；公式因子常量在 `model/skill_progression.go` 等 |
| 新指令 | `protocol`（如需）→ `app/command_registry.go` + handler → `engine/workflow*.go` → `manager` 新方法 → 前端 `messageText.js`、composable、GameView 装配 |
| 世界内容 | `worlds/*.json`、`npcs.json`、`dialogues.json` |
| UI | `components/*.vue`（结构与 scoped 样式）、`composables/*.js`（逻辑）、GameView（装配/布局样式） |

### 第四步：测试（分层门禁，全部通过才算完成）

```sh
# 1) 后端静态检查 + 全量测试（含资源一致性校验，任何 resources/ 改动都会被以下测试拦截）
cd MudGame-server
go build ./... && go vet ./... && go test ./...
#   - gamedata: TestStaticResourcesAreConsistent（全资源交叉引用校验、房间出口/出生点校验）
#   - engine/runtime: 副本模板与设计一致性、机关状态机全状态分类、剧情对话覆盖校验
#   - 注意：go test -race 在当前 Windows 机器不可用（ThreadSanitizer 分配失败），并发正确性靠锁纪律与评审

# 2) 前端
cd MudGame
npm run lint && npm run build
```

3) **冒烟测试**（涉及服务端行为时必须）：临时目录+临时端口起服，用 Node WebSocket 脚本走真实消息流：

```sh
MUD_DATA_DIR=$(mktemp -d) PORT=14009 go run ./cmd/mudserver   # 后台
node smoke.js   # 注册 → 登录 → 创角 → 进入游戏 → 执行目标行为 → 断言响应
```

4) **浏览器 E2E**（涉及 UI 时必须）：`npm run dev` 起前端，浏览器操作验证（截图/computed-style 探测），确认样式与交互无损。

5) **收尾**：杀掉测试进程、删除临时数据目录与脚本，不留杂物。

### 第五步：文档留存与提交

1. **回写文档**：实现与设计如有偏差，以最终实现回写权威文档；新副本更新玩家手册事实页并核对推荐入口；新文档在 README.md 索引登记；删除功能时**同步清理引用它的文档**（防止过期文档误导后续开发）。
2. **提交**（单仓库，一次特性一笔提交，文档与代码天然原子）：

```sh
git add -A && git commit -m "xxx"
```

3. 提交信息用中文短句、动词开头（例：“将NPC交谈改为消息区逐句展示”、“开放战斗中使用药品”）。**不主动 push**，推送时机由维护者决定。
4. 大改动前确认工作区干净，保证有可回退基线。

## 7. 常见陷阱

- **composables 互相 import 会被架构破坏**——永远通过 GameView 装配注入。
- **父组件 scoped CSS 打不进子组件内部**——样式跟着组件走。
- **`resources/` 改完必须跑 `go test ./...`**——孤儿引用、悬空出口、机关死状态都会被测试拦下。
- **改 NPC 模板的商店库存不影响已生成的运行时库存**（隔离副本机制）。
- **`utils/ws.js` 改动要保留心跳、重连与会话恢复逻辑**，注意 `event.target !== socket` 过期连接守卫。
- **测试一律用临时 MUD_DATA_DIR + 独立端口**，不要碰 `data/servers/server1` 正式数据。
- Windows 环境下 `go test -race` 不可用（环境限制，非代码问题）。
