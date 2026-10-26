# 后端安全部署与称号路径修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 部署时构建最新 Linux 后端，保留线上存档，校验后切换并支持失败回滚，修复跨机器部署后的称号读取。

**Architecture:** `deploy.bat` 只启动 PowerShell 7；`scripts/deploy.ps1` 负责检查、构建、清单与上传；Python 标准库脚本负责远端校验、备份、切换与回滚。称号读取复用 `gamedata.ResolveResourceDir()`。本次在当前会话内完成实现与验证，不部署正式服务。

**Tech Stack:** PowerShell 7、Python 3.10+、Go 1.26.1、OpenSSH、tar、Linux systemd。

## Global Constraints

- 服务：`oracle` 上的 `mud`；默认根目录 `/home/ubuntu/MudGame`，端口 4009。
- 正式存档保持在原位置；部署包不得包含本地 `data/`。
- 上传和校验失败时不停止正式服务；切换失败时恢复原程序和资源，不自动回写旧存档。
- Windows 使用 PowerShell 7、UTF-8、LiteralPath；所有测试使用临时存档和独立端口或隔离网络。
- 当前并发不超过 5 人，不重构性能架构，不修改游戏数值。

### Task 1: 运行时称号配置

**Files:** `MudGame-server/internal/manager/runtime_manager_titles.go`、`runtime_manager_titles_test.go`。

**Interfaces:** `loadTitleDefinitions() (map[string]string, error)` 通过 `gamedata.ResolveResourceDir()` 读取 `titles.json`；`getPlayerTitles` 对外行为保持一致。

- [x] 增加回归测试：自定义资源目录含另一份称号配置时，读取该目录的配置；缺失或损坏的配置返回错误。
- [x] 先运行 `go test ./internal/manager -run 'Title' -count=1` 确认测试揭示缺失接口。
- [x] 用资源目录解析器替换 `runtime.Caller`，保留一次加载和失败日志。
- [x] 同一测试通过；用跨平台构建、临时存档和真实 WebSocket 验证称号出现在角色属性中。

### Task 2: 构建与部署事务

**Files:** `deploy.bat`、`scripts/deploy.ps1`、`scripts/deploy_remote.py`、`scripts/tests/test_deploy_remote.py`。

**Interfaces:** `deploy.ps1 -BuildOnly` 仅生成本地发布包；`-StageOnly` 仅上传校验；默认部署。远端脚本接受根目录、发布 ID、服务名、程序名和健康检查端口；事务入口可注入服务操作与健康检查以进行隔离测试。

- [x] 用 Python `unittest` 覆盖完整切换、存档保留、切换失败回滚、校验失败不停服、失败停止新进程后再回滚、拒绝非法发布包。
- [x] 实现 PowerShell 参数验证、原生命令退出码检查、`go test`/`go vet`、Linux amd64 `go build -trimpath`，记录 Git 版本和文件 SHA-256；构建环境变量用 `finally` 恢复。
- [x] 发布包只包含新程序、`resources/` 和版本清单；所有上传在停服前完成。
- [x] 远端验证包路径与清单、systemd 工作目录和程序路径、资源目录与端口；保留部署锁，避免并发切换。
- [x] 停服并确认退出后备份存档、移动旧程序目录、切换新目录、启动及健康检查；失败停止新服务并恢复原目录再启动，保留失败包和备份供排查。
- [x] 执行 `python -m unittest discover -s scripts/tests -v` 和 PowerShell 语法解析；运行真实 `-BuildOnly`，检查包内版本、ELF 平台与清单。

### Task 3: 文档、Linux 验证与交付

**Files:** `docs/DEVELOPMENT.md`、`docs/SYSTEMS.md`、`docs/README.md`、`MudGame-server/README.md`。

- [x] 文档记录安全部署流程、BuildOnly/StageOnly 参数、备份和回滚位置、存档保留边界、运行时称号路径。
- [x] 后端执行 `go build ./...`、`go vet ./...`、`go test ./...`。
- [x] Linux 隔离环境执行部署事务测试，并以临时端口/存档做注册、登录、创角、选角、属性查询冒烟；验证称号加载日志。
- [x] 核对正式服务 PID 和健康检查未变化，清理本次临时进程与文件；回写计划勾选状态。
- [x] 只提交本次代码和文档，中文提交信息，不推送。

## 验证记录（2026-10-26）

- Go 全量 build/vet/test 通过，称号目录回归通过。
- 9 项 Python 部署事务测试在 Windows 和远端 Python 3.10 均通过；PowerShell 语法解析通过。
- `-BuildOnly` 真实构建发布包通过；`-StageOnly` 已在实际目标服务器完成上传校验和 systemd 预检，未停服。
- Linux 隔离网络冒烟通过：注册、登录、创角、选角、房间、属性、存档重载、称号“荒庙义士”和 `loaded 33 titles` 日志。
- Linux 隔离目录实际运行旧线上二进制后切换新版：账号和角色保留、落盘快照存在；故意让新版本健康检查失败后成功停止新进程、恢复旧发布并再次登录。
- 正式服务未切换；临时 Linux 测试进程和文件已清理。本地 `artifacts/backend/` 是刻意保留、Git 忽略的可检查构建产物。
