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

## 构建

```sh
npm run build
```

## 代码检查

```sh
npm run lint
```
