# Miniflux Indexer

[中文版](./README-zh.md)

- [Deployment](#deployment)
- [Configurations](#configurations)

Index [miniflux](https://miniflux.app/) entries with [sego](https://github.com/huichen/sego/) for better
full-text search support.

## Deployment

Requirements:

- PostgresQL >= 9.6

First, you should check your miniflux database and ensure it's
[Collate](https://www.postgresql.org/docs/current/collation.html) is `C.UTF-8` by using the following command:

```bash
psql -l
```

The output likes the following table:

```text
                                      List of databases
       Name       |    Owner    | Encoding | Collate |  Ctype  |      Access privileges
------------------+-------------+----------+---------+---------+-----------------------------
 miniflux         | miniflux    | UTF8     | C.UTF-8 | C.UTF-8 |
------------------+-------------+----------+---------+---------+-----------------------------
```

If it's not `C.UTF-8`, you should convert it to `C.UTF-8` or the indexer won't work.

Then you need to create a table named `index_info` for storing indexer-related information. It's highly 
recommended using a separate database or schema from miniflux.

You may create a new database using the following command:

```bash
createdb -O miniflux miniflux_indexer
```

Then use [migrate](https://github.com/golang-migrate/migrate/) to execute migrations:

```bash
migrate -database $DATABASE_URL -path ./migrations up
```

Then you can get miniflux-indexer running by using the following command:

```bash
miniflux-indexer start --database-url $DATABASE_URL --miniflux-database-url $MINIFLUX_DATABASE_URL
```

The first run may consume more than 1 GB of memory, please ensure you have enough memory.

## Configurations

Miniflux-indexer can be configured via environment variables or command line arguments. The configuration options
can be viewed by using `miniflux-indexer start --help` and `miniflux-indexer --help`.

Here is a sample `.env` file:

```bash
DATABASE_URL='postgres://miniflux:miniflux@127.0.0.1/miniflux_indexer?sslmode=disable'
MINIFLUX_DATABASE_URL='postgres://miniflux:miniflux@@127.0.0.1/miniflux?sslmode=disable'
INDEXER_BATCH_SIZE=50
LOG_LEVEL=info 
LOG_FORMAT=human
# extra configuration for controlling database driver logging, default is warn
PGX_LOG_LEVEL=error
```

More database driver options can be specified by add query parameters to database urls, including:

- `pool_max_conns`: integer greater than 0
- `pool_min_conns`: integer 0 or greater
- `pool_max_conn_lifetime`: duration string
- `pool_max_conn_idle_time`: duration string
- `pool_health_check_period`: duration string

Check [pgxpool documentation](https://github.com/jackc/pgx/blob/master/pgxpool/pool.go#L254) for more info.
