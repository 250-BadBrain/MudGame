# MudGame Go Server (Migration)

这是 MudGame 后端从 Java 迁移到 Go 的工程。

## 当前分层（运行时内存化后）

- `cmd/mudserver`：启动入口
- `internal/app`：网络层（WebSocket 接入、会话、事件广播）
- `internal/engine`：业务编排层（对外服务接口与流程编排）
- `internal/manager`：运行时管理层（玩家/房间/副本等动态状态的内存维护与指令处理）
- `internal/storage`：持久化层（仅负责动态数据与磁盘之间的读写）
- `internal/model`：领域模型定义
- `internal/protocol`：协议结构（CommandMessage/GameResult）

## 运行时 I/O 策略

- 启动阶段：一次性加载静态资源与动态数据到内存。
- 运行阶段：前端所有指令仅在内存态处理，不执行磁盘读写。
- 持久化阶段：仅通过定时落盘（autosave）与进程退出时 flush 将动态数据写回磁盘。

## 当前已迁移内容

- WebSocket 入口: /ws
- 健康检查: /health
- 协议结构: CommandMessage / GameResult
- system 指令主流程（注册/登录/删号/创角/选角/重连/房间）
- command 指令主流程（状态/背包/技能/装备/世界/移动/商店/学习/挖矿/战斗/复活）
- chat 公开聊天
- event 广播（player_left/player_entered/player_status_changed/room_update/room_message）

## 运行

在 MudGame-server 目录执行:

go run ./cmd/mudserver

默认端口 4009，可通过 PORT 环境变量覆盖。

默认动态数据目录为 `../data/servers/server1`，可通过 `MUD_DATA_DIR` 覆盖。
静态资源默认从 `resources/` 加载，可通过 `MUD_RESOURCES_DIR` 覆盖。

## 存储说明

当前版本仅保留文件存储实现，不再包含内存存储后端。
