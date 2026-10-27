# MudGame

MudGame 前端仓库，基于 Vue 3、Vite、Pinia 和 Vue Router。

游戏内容、新手路线、全部副本攻略与剧情见 [玩家手册](../docs/player/README.md)。

统一开发流程（模块职责、标准开发步骤、测试门禁）见根目录 `../docs/DEVELOPMENT.md`。前端两条硬性约定：composables 之间禁止互相 import（由 `GameView.vue` 装配注入）；父组件 scoped CSS 只能命中子组件根元素。

## 开发

```sh
npm install
npm run dev
```

默认会连接当前主机的 4009 端口 WebSocket 服务。

如需显式指定后端地址，可创建 `.env.local`：

```sh
VITE_WS_URL=ws://localhost:4009/ws
```

如果前端部署在 Cloudflare Pages（`mud.game.h2seo4.win`），而后端运行在你本机并通过 Cloudflare Tunnel 暴露到 `api-mud.h2seo4.win`，请在生产环境使用 `.env.production`：

```sh
VITE_WS_URL=wss://api-mud.h2seo4.win/ws
```

Cloudflare Pages 默认以站点根路径部署，因此不需要额外的 Vite `base` 配置。

## 构建

```sh
npm run build
```

## Cloudflare Pages + Tunnel 部署要点

1. 前端部署到 Cloudflare Pages，构建命令为 `npm run build`，输出目录为 `dist`。**monorepo 注意**：仓库根目录是整个项目，Pages 项目设置中需把 **Root directory（高级）设为 `MudGame`**；并在 **Settings → Builds & deployments → Build watch paths** 配置包含路径 `/MudGame/**`——仅设根目录不会自动跳过无关提交，需 watch paths 才能让只改后端/文档的 push 不触发前端重建。
2. 本机启动后端：在仓库根目录运行 `start.bat`（等价于在 `MudGame-server` 下执行 `go run ./cmd/mudserver`）。
3. 用 Cloudflare Tunnel 将本机 4009 暴露到公网域名。
4. 在 Cloudflare Pages 环境变量中设置 `VITE_WS_URL=wss://api-mud.h2seo4.win/ws` 后重新部署。

## 代码检查

```sh
npm run lint
```
