# 破碎江湖

黄易武侠背景的网页文字游戏：以洛阳等公共城市为活动中心，通过交谈、修行、探索、副本战斗和机关推进故事，取得武学传承、装备与称号。

## 我现在可以玩什么

从 **[玩家手册：现在可以玩什么](docs/player/README.md)** 开始。它包含新手路线、七卷故事、全部副本的触发与逐房攻略、剧情剧透、八城地图、任务和收藏来源。

- [78 个副本的解锁顺序](docs/player/unlock_order.md)
- [世界任务怎么接、怎么交](docs/player/quests.md)
- [老师、商店与城市交谈](docs/player/worlds.md)
- [秘籍、装备、物品与称号](docs/player/collections.md)
- [当前尚未接通的内容](docs/player/content_status.md)

第一次游玩可先做「洛水传闻」，去武馆和书院修行，再打「荒庙夜雨」与「扬子江惊变」。手册会区分已接通任务、剧情伏笔和仅有配置的内容。

## 开发与运行

- [统一开发指导](docs/DEVELOPMENT.md) · [系统总览](docs/SYSTEMS.md) · [文档索引](docs/README.md)
- 前端：`MudGame/`，Vue 3 + Vite；在该目录运行 `npm install`、`npm run dev`。
- 后端：`MudGame-server/`，Go + WebSocket；在该目录运行 `go run ./cmd/mudserver`，默认端口 4009。
- 部署：根目录 `deploy.bat`；`-BuildOnly` 只构建，`-StageOnly` 只上传校验。备份、存档保留和回滚规则见开发指导。

更新玩家事实页：`python scripts/generate_player_guide.py`；检查覆盖和链接：`python scripts/generate_player_guide.py --check`。
