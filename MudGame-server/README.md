# MudGame Go Server

破碎江湖后端服务（Go + WebSocket）。开发流程、任务清单与测试门禁见根目录 `../docs/DEVELOPMENT.md`。

## 分层

- `cmd/mudserver`：启动入口
- `internal/app`：网络层（WebSocket 接入、会话、命令注册与分发、事件广播、错误本地化）
- `internal/engine`：业务编排层（对外服务接口与流程编排；`engine/runtime` 为副本实例管理）
- `internal/manager`：运行时管理层（玩家/房间/NPC/战斗/副本/商店/学习等动态状态的内存维护与指令实现）
- `internal/storage`：持久化层（仅负责动态数据与磁盘之间的读写，原子写 + 定时落盘）
- `internal/model`：领域模型与数值公式
- `internal/gamedata`：静态资源加载与一致性校验
- `internal/protocol`：协议结构（CommandMessage/GameResult）

## 运行时 I/O 策略

- 启动阶段：一次性加载静态资源（`resources/`）与动态数据到内存。
- 运行阶段：前端所有指令仅在内存态处理，不执行磁盘读写。
- 持久化阶段：仅通过定时落盘（autosave）与进程退出时 flush 将动态数据写回磁盘。

## 功能范围

- WebSocket 入口 `/ws`，健康检查 `/health`
- system 指令：注册/登录/删号/创角/选角/重连
- command 指令：状态/背包/物品使用/技能学习/装备/世界与移动/采集与帮工/商店/NPC 对话/副本（进入、目标、机关、结算）/战斗与切磋/复活（虚弱）
- chat 公开聊天
- event 广播（player_left/player_entered/player_status_changed/room_update/room_message 等）

## 运行

在 MudGame-server 目录执行:

go run ./cmd/mudserver

默认端口 4009，可通过 PORT 环境变量覆盖。

默认动态数据目录为 `../data/servers/server1`，可通过 MUD_DATA_DIR 覆盖。
静态资源默认从 `resources/` 加载，可通过 MUD_RESOURCES_DIR 覆盖。
称号配置 `titles.json` 使用相同的资源目录解析，不依赖编译机器上的源码路径，支持 `-trimpath` 和跨机器部署。
**测试时请使用临时数据目录与独立端口**，避免污染正式数据。

## 部署

从仓库根目录运行 `deploy.bat`：先执行后端测试与静态检查，再从当前源码编译 Linux amd64 程序，上传完整校验包后停服、备份和切换。保留远端存档；启动或健康检查失败时回滚程序和资源。

`deploy.bat -BuildOnly` 仅在本地生成发布包；`deploy.bat -StageOnly` 增加远端校验，仍不停服。默认配置、依赖、参数、备份和故障恢复规则见 [统一开发指导文档](../docs/DEVELOPMENT.md#后端部署windows--linux)。

## 测试

go build ./... && go vet ./... && go test ./...

注意：`go test -race` 在当前 Windows 开发机不可用（ThreadSanitizer 分配失败，环境限制）。
