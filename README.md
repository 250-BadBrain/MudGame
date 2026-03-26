# MudGame

MudGame 前端仓库，基于 Vue 3、Vite、Pinia 和 Vue Router。

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

如果前端部署在 Cloudflare Pages（`mud.game.h2seo4.win`），而后端运行在你本机并通过 Cloudflare Tunnel 暴露到 `api-mud.game.h2seo4.win`，请在生产环境使用 `.env.production`：

```sh
VITE_WS_URL=wss://api-mud.game.h2seo4.win/ws
```

Cloudflare Pages 默认以站点根路径部署，因此不需要额外的 Vite `base` 配置。

## 构建

```sh
npm run build
```

## Cloudflare Pages + Tunnel 部署要点

1. 前端部署到 Cloudflare Pages，构建命令为 `npm run build`，输出目录为 `dist`。
2. 本机启动后端：`mvn spring-boot:run`。
3. 用 Cloudflare Tunnel 将本机 4009 暴露到公网域名。
4. 在 Cloudflare Pages 环境变量中设置 `VITE_WS_URL=wss://api-mud.game.h2seo4.win/ws` 后重新部署。

## 代码检查

```sh
npm run lint
```
