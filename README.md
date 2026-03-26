# MudGame-server

MudGame 后端仓库，基于 Spring Boot 和 WebSocket。

## 运行要求

- Java 21
- Maven 3.9+

## 启动

```sh
mvn spring-boot:run
```

默认监听 4009 端口，WebSocket 地址为 `/ws`。

如果前端部署在 Cloudflare Pages 的 `mud.game.h2seo4.win`，而后端运行在本机，可通过 Cloudflare Tunnel 暴露后端端口到 `api-mud.game.h2seo4.win`。对应 WebSocket 地址应写为 `wss://api-mud.game.h2seo4.win/ws`。

说明：本机 Spring Boot 仍然监听普通的 `ws` 连接，TLS 由 Cloudflare 终止并转换为公网侧的 `wss`。这就是 Cloudflare Pages 的 `https` 页面能够连接你本机后端的正确方式。

默认允许的前端来源包含本地开发地址、`https://*.pages.dev` 和 `https://mud.game.h2seo4.win`。如需精确限制来源，可在运行时设置环境变量 `MUD_ALLOWED_ORIGIN_PATTERNS` 覆盖默认值。

## 打包

```sh
mvn clean package
```

## 运行数据

- 账号与角色存档位于 `data/`
- 运行日志位于 `logs/`
- 游戏静态资源与世界数据位于 `src/main/resources/`