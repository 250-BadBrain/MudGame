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

## 打包

```sh
mvn clean package
```

## 运行数据

- 账号与角色存档位于 `data/`
- 运行日志位于 `logs/`
- 游戏静态资源与世界数据位于 `src/main/resources/`