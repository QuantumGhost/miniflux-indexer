# Miniflux Indexer

[English Version](./README.md)

- [部署指南](#部署指南)
- [配置](#配置)

使用 [sego](https://github.com/huichen/sego/) 为 [Miniflux](https://miniflux.app/) 
条目创建索引，以更好地支持全文搜索。

## 部署指南

部署需求：

- PostgresQL >= 9.6

首先，你需要检查你的 miniflux 数据库，确保 [COLLATE](https://www.postgresql.org/docs/current/collation.html) 是 `C.UTF-8`，你可以使用下面的命令来检查：

```bash
psql -l
```

输出类似下表：

```text
                                      List of databases
       Name       |    Owner    | Encoding | Collate |  Ctype  |      Access privileges
------------------+-------------+----------+---------+---------+-----------------------------
 miniflux         | miniflux    | UTF8     | C.UTF-8 | C.UTF-8 |
------------------+-------------+----------+---------+---------+-----------------------------
```

如果 Collate 不是 `C.UTF-8`，你应该先转换到 `C.UTF-8`，否则 `miniflux-indexer` 不会正常工作。

然后需要创建一个名为 `index_info` 的数据库表来存储 `miniflux-indexer` 相关的信息，这里强烈使用
与 `miniflux` 不同的数据库或[模式](https://www.postgresql.org/docs/current/ddl-schemas.html)。

采用下面的命令来创建数据库：

```bash
createdb -O miniflux miniflux_indexer
```

然后使用 [migrate 工具](https://github.com/golang-migrate/migrate/) 来执行数据库迁移脚本：

```bash
migrate -database $DATABASE_URL -path ./migrations up
```

然后，用下面的命令来启动 `miniflux-indexer`：

```bash
miniflux-indexer start --database-url $DATABASE_URL --miniflux-database-url $MINIFLUX_DATABASE_URL
```

第一次索引可能会需要 1GB 内存，请确保内存充足。

## 配置

`miniflux-indexer` 可以使用[环境变量](https://zh.wikipedia.org/zh-cn/%E7%8E%AF%E5%A2%83%E5%8F%98%E9%87%8F) 或命令行参数进行设置，设置项可以通过 
`miniflux-indexer start --help` 和 `miniflux-indexer --help` 命令来查看。

下面是一个示意的 [.env]() 配置文件：

```bash
DATABASE_URL='postgres://miniflux:miniflux@127.0.0.1/miniflux_indexer?sslmode=disable'
MINIFLUX_DATABASE_URL='postgres://miniflux:miniflux@@127.0.0.1/miniflux?sslmode=disable'
INDEXER_BATCH_SIZE=50
LOG_LEVEL=info 
LOG_FORMAT=human
# 控制数据库驱动日志级别的环境变量，默认为警告级别
PGX_LOG_LEVEL=error
```

更多的数据库驱动配置可以在数据 URL 上添加查询参数指定，包含下面设置项：

- `pool_max_conns`: integer greater than 0
- `pool_min_conns`: integer 0 or greater
- `pool_max_conn_lifetime`: duration string
- `pool_max_conn_idle_time`: duration string
- `pool_health_check_period`: duration string

详情请参考 [pgxpool 文档](https://github.com/jackc/pgx/blob/master/pgxpool/pool.go#L254)。
