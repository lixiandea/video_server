# P2-5 版本化数据库迁移策略

> 目标（路线图 P2-5 验收标准）：**迁移可回滚，生产不再 `AutoMigrate`**。
> 本文是实施前的设计说明，实施按第 8 节的四步拆分提交。

## 1. 现状与问题

| 项 | 现状 |
| --- | --- |
| 建表方式 | 三个服务（api-server / scheduler / transcode-worker）启动时各自调用 `database.InitDatabase` → `migrateSchema` 逐表 `AutoMigrate`，用 MySQL `GET_LOCK` 串行化（P1-8/P1-9 的产物） |
| 手工 DDL | `docker/mysql/init.sql` 只建库 + `users` 表 + admin 种子账号 |
| 版本管理 | **无**。没有版本表，无法知道某个库处于哪个版本，也没有回滚手段 |
| 真实库实测（`video_server`，MySQL 8.0.46） | 6 张表：`users`(37) / `videos`(25) / `comments`(32) / `sessions`(4) / `video_likes`(0) / `video_deletion_records`(4) |

**已存在的漂移证据**（说明「模型即真相」并不可靠）：

1. `users` 上同时存在两个等价唯一索引：`uni_users_login_name`（init.sql 建的）与
   `idx_users_login_name`（GORM 按 `uniqueIndex` 又建了一个）。
2. `users.pwd` 在 init.sql 里声明为 `VARCHAR(255)`，实际是 `longtext`（AutoMigrate 放宽所致）；
   `videos.name`/`file_path`、`comments.content`、`display_c_time`、`reason` 同样是 `longtext`。
3. `sessions.ttl` 列名与 Go 字段 `ExpiresAt` 不一致（P1-7 用 `gorm:"column:ttl"` 保留兼容）。

结论：迁移基线必须**以真实库结构为准**（而非以 `internal/models` 反推），并在清理漂移时保持幂等。

## 2. 设计总览

采用「版本化 SQL 迁移 + 显式执行 + 版本表」：

```
migrations/
├── 0001_init_schema.up.sql / .down.sql        # 基线：6 张表的 CREATE TABLE IF NOT EXISTS
├── 0002_dedupe_users_unique_index.up.sql / .down.sql
└── 000N_<snake_case>.up.sql / .down.sql        # 后续每个变更一组
```

- 文件名规范：`<4 位序号>_<snake_case 描述>.{up,down}.sql`；序号唯一、按字典序执行。
- 脚本内**不写** `CREATE DATABASE` / `USE`；目标库由连接串决定，便于在临时库上做验证。
- 每个 `up` 必须配套 `down`；`0001` 除外（见第 6 节的基线保护）。

### 版本表

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version     VARCHAR(20)  NOT NULL,       -- 0001
  name        VARCHAR(191) NOT NULL,       -- init_schema
  checksum    CHAR(64)     NOT NULL,       -- SHA-256(up 文件内容)
  applied_at  DATETIME(3)  NOT NULL,
  duration_ms INT          NOT NULL,
  PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`checksum` 用于防止「已应用的迁移文件被事后修改」：一旦不匹配，执行器直接报错退出，
而不是让库结构与仓库里的脚本悄悄分叉。

## 3. 执行器（`pkg/database/migrator.go`）

```
migrate up      → 依次应用所有 pending 版本
migrate down N  → 倒序回滚最近 N 个版本（需 --yes 确认）
migrate status  → 打印全部版本：applied / pending / checksum mismatch
```

关键行为：

1. **锁**：进入迁移前 `SELECT GET_LOCK('video_server:migrate', 30)`，整段执行期间持有，
   结束后 `RELEASE_LOCK`；拿不到锁则退出并提示「另一个实例正在迁移」。沿用 P1-9 的串行化思路。
2. **按文件登记**：单个版本文件内的语句逐条 `Exec`，**全部成功后**才写入 `schema_migrations`。
3. **失败处理**：MySQL 的 DDL 会隐式提交、无法用事务包裹，因此失败时版本不会被登记；
   运维修复后重跑即可。规范要求：**每个版本尽量只做一件事**，避免一次失败留下多个半成品。
4. **语句切分**：按「行首 `--` 为注释、语句以 `;` 结尾」切分；禁止在字符串字面量中出现 `;`（写入规范并由单测覆盖）。
5. **幂等优先**：`CREATE TABLE IF NOT EXISTS`、`INSERT IGNORE`；
   需要条件判断的 DDL（MySQL 8 不支持 `ADD COLUMN IF NOT EXISTS`）使用
   `SET @sql = IF(EXISTS(...), '...', 'DO 0'); PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;`
   惯用法，保证同一版本可重复执行。

## 4. 服务侧改动（去掉启动时 AutoMigrate）

| 位置 | 改动 |
| --- | --- |
| `pkg/database.InitDatabase` | 默认**不再**调用 `migrateSchema` |
| 新增开关 `DB_AUTO_MIGRATE` | 默认 `false`；置 `true` 时**仅在非 release 模式**生效（release 下忽略并告警），避免生产误开 |
| 新增 `cmd/migrate` | 子命令 `up` / `down --steps N` / `status`；`./build.sh` 一并产出 `bin/migrate` |
| `docker-compose.yml` | 新增一次性服务 `migrate`（`restart: "no"`）；给 `mysql` 补 `healthcheck`；api-server / scheduler / transcode-worker 改为 `depends_on: mysql(service_healthy) + migrate(service_completed_successfully)` |
| `docker/mysql/init.sql` | 精简为**只建库**；表结构交给 `0001`，admin 种子账号移除（见第 9 节决策 A） |
| `Makefile` | 新增 `migrate-up` / `migrate-down` / `migrate-status` 目标 |

## 5. 基线如何接管既有数据库

`0001_init_schema.up.sql` 全部使用 `CREATE TABLE IF NOT EXISTS`（结构取自真实库 `SHOW CREATE TABLE`）：

- **既有库**（当前由 AutoMigrate 建起来的）：语句全部空转，随后登记 `0001` → 平滑接管，**数据零变化**；
- **新库**：直接建出与现状一致的表，不再依赖 GORM；
- 基线里保留 init.sql 建立的索引名（`uni_users_login_name` 等），避免影响既有库。

## 6. 回滚策略

| 层级 | 手段 | 约束 |
| --- | --- | --- |
| 单版本 | `migrate down --steps 1 --yes` | 执行该版本 `.down.sql`，成功后删除对应版本行 |
| 多版本 | `migrate down --steps N --yes` | 倒序逐版本执行，任一步失败立即停止并报错（不做部分回滚） |
| 基线 `0001` | **默认拒绝执行** | 其 down 会删除全部业务表，仅在显式设置 `MIGRATE_ALLOW_BASELINE_DOWN=1` 时允许 |
| 灾难恢复 | `mysqldump` 备份 | 上线前必做：`docker compose exec -T mysql mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" video_server > backup_$(date +%F).sql` |
| 已应用的 down 脚本 | `checksum` 校验 | 与 up 同规则：文件被改过（累计 checksum）即拒绝执行 |

规范：`down` 必须与 `up` 对称，且能在「该版本已应用」的状态下**重复执行不报错**。

## 7. 验收标准

| # | 场景 | 期望 |
| --- | --- | --- |
| 1 | 空库 `migrate up` | 建出 6 张表，`schema_migrations` 记录全部版本；与既有库的 `SHOW CREATE TABLE` 归一化后一致 |
| 2 | 既有库（AutoMigrate 产物，含数据）`migrate up` | 表结构与数据零变化，仅新增 `schema_migrations` 与其记录 |
| 3 | 连续两次 `migrate up` | 第二次输出「no pending migrations」并退出码 0 |
| 4 | 并发两个 `migrate up` | 只有一个执行，另一个明确提示锁被占用（不产生重复 DDL） |
| 5 | 对 `0002` 做 `up` → `down` | 列/索引回退，版本行删除，可重复执行 |
| 6 | 摘除 AutoMigrate 后启动 api-server | 表结构不变（证明服务不再建表） |
| 7 | 文件被改后 `migrate status` | 报 `checksum mismatch` 并给出提示 |

## 8. 实施顺序（四步拆分提交）

1. **执行器与 CLI（不改变现有行为）**：`pkg/database/migrator.go` + 单测（切分、排序、
   checksum、锁、条件 DDL）+ `cmd/migrate`；`DB_AUTO_MIGRATE` 默认 `true` 保持现状。
2. **迁移脚本**：`0001_init_schema`（取自真实库结构）与 `0002_dedupe_users_unique_index`；
   在临时库与既有库分别验证（验收 #1/#2）。
3. **切换默认**：`DB_AUTO_MIGRATE=false`、compose 增加 `migrate` 一次性服务与 mysql healthcheck、
   `init.sql` 精简、`./build.sh` 产出 `bin/migrate`、Makefile 目标、文档与 AGENTS 同步。
4. **一致性门禁（可选）**：CI 中在临时库跑 `migrate up`，再用 GORM 的
   `Migrator().HasTable/HasColumn` 断言模型与库结构一致——防止后续改了 `internal/models`
   却忘记写迁移。

## 9. 待确认的决策点

| # | 决策 | 建议 |
| --- | --- | --- |
| A | `docker/mysql/init.sql` 里的 admin 种子账号（含 bcrypt 散列）是否保留 | **删除**：口令散列不应入库版本控制（P0-1），改为文档指引用 `POST /api/v1/users/register` 创建 |
| B | `users` 上重复的唯一索引是否在 `0002` 清理 | **是**：保留 `uni_users_login_name`，用条件 DDL 删除 `idx_users_login_name` |
| C | 迁移执行方式 | **一次性 compose 服务**（`migrate`）+ 显式 `make migrate-up`；不建议让 api-server 启动时自动迁移 |
| D | 是否保留 `DB_AUTO_MIGRATE` 开关 | **保留**（默认 `false`），便于本地/CI 快速起库，但 release 模式下强制忽略 |

## 10. 风险与缓解

| 风险 | 缓解 |
| --- | --- |
| 基线 DDL 与 `internal/models` 漂移（改了模型忘写迁移） | 第 8.4 步的 CI 一致性断言；`checksum` 防篡改 |
| MySQL DDL 非事务，失败留半成品 | 每版本只做一件事；失败不登记版本 + 文档化恢复步骤；上线前备份 |
| 脚本切分误判（注释、字符串内分号） | 明确规范 + 单测覆盖切分逻辑 |
| 三服务并发迁移 | 整段持 `GET_LOCK`；编排上改为一次性 `migrate` 服务从根上避免并发 |
| 既有数据丢失 | 基线只 `CREATE TABLE IF NOT EXISTS`、绝不 `DROP`；`down` 默认禁止且需显式确认 |
| 迁移把大表锁住影响线上 | 单版本单 DDL、必要时用 `ALGORITHM=INPLACE`；避开业务高峰执行 |

---

## 11. 实施记录（2026-10-01）

决策点 A–D 均按建议执行：删除 admin 种子账号、清理重复唯一索引、使用一次性
compose 服务执行迁移、保留 `DB_AUTO_MIGRATE` 开关（默认关闭，release 强制忽略）。

按第 8 节的四步落地（前三步已完成，第四步为可选门禁）：

| 步骤 | 提交 | 内容 |
| --- | --- | --- |
| 1 | `e6c49b9` | `pkg/database/migrator.go` + `cmd/migrate` + 7 组单测；`Connect` 从 `InitDatabase` 拆出；`build.sh` 产出 `bin/migrate` |
| 2 | `e0aef29` | `0001_init_schema`（真实库结构）与 `0002_dedupe_users_unique_index`；修复验证中发现的「部分回滚」缺陷 |
| 3 | 本步 | `DB_AUTO_MIGRATE` 默认关闭（release 强制忽略）、`init.sql` 只建库、`Dockerfile.migrate` 与一次性 `migrate` 服务、mysql `healthcheck`、三个服务改为等待迁移完成、Makefile 迁移目标 |

### 验证结果

| # | 场景 | 结果 |
| --- | --- | --- |
| 1 | 空库 `migrate up` | ✅ 6 张业务表 + 版本表，与既有库逐表 `SHOW CREATE TABLE` 完全一致 |
| 2 | 既有库（含数据）`migrate up` | ✅ 除 0002 的索引清理外结构与数据零变化（逐表 diff + 行数比对） |
| 3 | 连续两次 `up` | ✅ 第二次输出 `no pending migrations` |
| 4 | 并发两个 `up` | ✅ 一个执行、另一个等锁后报 `no pending`，无重复 DDL |
| 5 | `down` 与基线保护 | ✅ 0002 可回滚且可重复；命中基线时整体拒绝，不产生部分回滚 |
| 6 | 服务不再建表 | ✅ 空库启动 api-server 后表数仍为 0；`DB_AUTO_MIGRATE=true` 时才建表 |
| 7 | checksum 防护 | ✅ 改动已应用脚本后 `status` 报 `checksum-mismatch`（退出码 1），`up` 直接拒绝 |
| 8 | 全新编排 | ✅ `down -v` 后一键启动：migrate `Exited(0)`、7 张表齐备、`test-api.sh` 13/13、Playwright e2e 19/19 |

### 已知限制

- MySQL 的 DDL 隐式提交，无法把整个版本包进事务；失败版本不会登记，修复后重跑即可，
  脚本规范要求单版本单变更 + 幂等写法。
- 基线 `0001` 显式指定 `utf8mb4_0900_ai_ci`，要求 MySQL 8.0（与 compose / CI 一致）。
