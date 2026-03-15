# 数据库Migration管理详解

## 🎯 核心问题

### 场景：你的项目上线运行中，现在要加一个字段

**没有Migration（你现在的做法）**：
```sql
-- 手动在生产数据库执行
ALTER TABLE users ADD COLUMN email VARCHAR(255);
```

**问题**：
1. 团队成员A的本地数据库有email字段，成员B没有 → 代码跑不起来
2. 测试环境、预发布环境、生产环境结构不一致
3. 不知道谁在什么时候改了数据库
4. 回滚代码后，数据库无法回滚 → 新字段留在那里
5. 新同事加入，不知道要执行哪些SQL

---

## ✅ Migration是什么？

**数据库版本管理工具**，像Git管理代码一样管理数据库结构。

```
代码版本: v1.0 → v1.1 → v1.2
数据库版本: migration_001 → migration_002 → migration_003
```

---

## 📂 Migration文件示例

### 结构
```
migrations/
├── 000001_create_users_table.up.sql     # 升级
├── 000001_create_users_table.down.sql   # 回滚
├── 000002_add_email_to_users.up.sql
├── 000002_add_email_to_users.down.sql
├── 000003_create_videos_table.up.sql
└── 000003_create_videos_table.down.sql
```

### 内容示例

**000001_create_users_table.up.sql**（升级）：
```sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(32) UNIQUE NOT NULL,
    password VARCHAR(128) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_name ON users(name);
```

**000001_create_users_table.down.sql**（回滚）：
```sql
DROP INDEX idx_users_name ON users;
DROP TABLE users;
```

**000002_add_email_to_users.up.sql**：
```sql
ALTER TABLE users ADD COLUMN email VARCHAR(255);
ALTER TABLE users ADD UNIQUE INDEX idx_users_email (email);
```

**000002_add_email_to_users.down.sql**：
```sql
ALTER TABLE users DROP INDEX idx_users_email;
ALTER TABLE users DROP COLUMN email;
```

---

## 🔄 工作流程

### 1. 记录当前版本

数据库中有张表记录当前执行到哪个migration：
```sql
CREATE TABLE schema_migrations (
    version BIGINT PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);

-- 示例数据
| version | dirty |
|---------|-------|
| 2       | false |  -- 当前版本是2，未损坏
```

### 2. 执行Migration

```bash
# 升级到最新版本
migrate -path ./migrations -database "mysql://user:pwd@tcp(localhost:3306)/video_server" up

输出：
  ✓ 000001_create_users_table.up.sql
  ✓ 000002_add_email_to_users.up.sql
```

**内部逻辑**：
1. 读取 `schema_migrations`，当前版本=2
2. 扫描 `migrations/` 目录，发现 000003 未执行
3. 执行 `000003_create_videos_table.up.sql`
4. 更新 `schema_migrations` 版本=3

### 3. 回滚Migration

```bash
# 回滚一个版本
migrate -path ./migrations -database "..." down 1

输出：
  ✓ 000003_create_videos_table.down.sql
```

---

## 🎬 实际场景演示

### 场景1：新功能开发

```bash
# 1. 开发人员在本地创建migration
$ migrate create -ext sql -dir migrations -seq add_avatar_to_users

生成文件：
  migrations/000004_add_avatar_to_users.up.sql
  migrations/000004_add_avatar_to_users.down.sql

# 2. 编写SQL
# 000004_add_avatar_to_users.up.sql
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(512);

# 000004_add_avatar_to_users.down.sql
ALTER TABLE users DROP COLUMN avatar_url;

# 3. 在本地执行
$ migrate -path ./migrations -database "..." up
  ✓ 000004_add_avatar_to_users.up.sql

# 4. 提交代码+migration文件到Git
$ git add migrations/000004*
$ git commit -m "feat: 添加用户头像字段"
```

### 场景2：其他成员拉取代码

```bash
# 1. 拉取代码
$ git pull

# 2. 自动执行migration（或CI/CD自动执行）
$ migrate -path ./migrations -database "..." up
  ✓ 000004_add_avatar_to_users.up.sql

# ✅ 数据库自动同步，无需手动执行SQL
```

### 场景3：部署到生产

```bash
# CI/CD流程
1. 构建镜像
2. 运行migration容器（先升级数据库）
   $ docker run --rm migrate/migrate \
       -path=/migrations \
       -database="mysql://..." \
       up

3. 部署应用容器（数据库已升级）
```

### 场景4：线上Bug，需要回滚

```bash
# 1. 回滚代码到上一个版本（v1.1 → v1.0）
$ git checkout v1.0

# 2. 回滚数据库（migration 4 → 3）
$ migrate -path ./migrations -database "..." down 1
  ✓ 000004_add_avatar_to_users.down.sql

# ✅ 代码和数据库都回到v1.0状态
```

---

## 🛠️ 常用工具

### 1. golang-migrate（推荐）

**安装**：
```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
mv migrate /usr/local/bin/
```

**使用**：
```bash
# 创建migration
migrate create -ext sql -dir migrations -seq create_comments_table

# 升级
migrate -path ./migrations -database "mysql://root:pwd@tcp(localhost:3306)/video_server" up

# 降级
migrate -path ./migrations -database "..." down 1

# 强制设置版本（慎用）
migrate -path ./migrations -database "..." force 3
```

### 2. GORM AutoMigrate（简单项目可用）

```go
// ❌ 不推荐：生产环境自动迁移
func main() {
    db.AutoMigrate(&User{}, &Video{}, &Comment{})
}

// 问题：
// 1. 只能添加字段，无法删除字段
// 2. 无法回滚
// 3. 没有版本记录
// 4. 索引管理不清晰
```

### 3. Flyway（Java生态常用）

### 4. Alembic（Python生态）

---

## 🎯 你的项目改进方案

### 当前问题

```sql
-- 现在的做法：手动创建表
CREATE TABLE users (...);
CREATE TABLE video_info (...);
CREATE TABLE comments (...);
```

**问题**：
1. 新同事加入，需要手动找SQL文件执行
2. 多个环境的表结构可能不一致
3. 表结构变更没有历史记录
4. 无法回滚

---

### 改进步骤

#### Step 1: 安装golang-migrate

```bash
# 方式1: 二进制安装
wget https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz
tar -xvzf migrate.linux-amd64.tar.gz
sudo mv migrate /usr/local/bin/

# 方式2: Go安装
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Step 2: 创建migrations目录

```bash
mkdir -p migrations
```

#### Step 3: 转换现有表结构

```bash
# 创建第一个migration
migrate create -ext sql -dir migrations -seq init_schema
```

**migrations/000001_init_schema.up.sql**：
```sql
-- 用户表
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(32) UNIQUE NOT NULL,
    password VARCHAR(128) NOT NULL,
    is_valid TINYINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_users_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 视频表
CREATE TABLE video_info (
    id INT AUTO_INCREMENT PRIMARY KEY,
    vid VARCHAR(64) UNIQUE NOT NULL,
    author_id INT NOT NULL,
    name VARCHAR(128) NOT NULL,
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    click_count INT DEFAULT 0,
    INDEX idx_video_vid (vid),
    INDEX idx_video_author (author_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 评论表
CREATE TABLE comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    comment_id VARCHAR(64) UNIQUE NOT NULL,
    video_id VARCHAR(64) NOT NULL,
    author_id INT NOT NULL,
    content TEXT NOT NULL,
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_comment_video (video_id),
    INDEX idx_comment_id (comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Session表
CREATE TABLE sessions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(128) UNIQUE NOT NULL,
    user_id INT NOT NULL,
    username VARCHAR(32) NOT NULL,
    ttl VARCHAR(32),
    INDEX idx_session_id (session_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 视频删除记录表
CREATE TABLE video_delete_record (
    id INT AUTO_INCREMENT PRIMARY KEY,
    vid VARCHAR(64) NOT NULL,
    INDEX idx_vdr_vid (vid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**migrations/000001_init_schema.down.sql**：
```sql
DROP TABLE IF EXISTS video_delete_record;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS video_info;
DROP TABLE IF EXISTS users;
```

#### Step 4: 执行Migration

```bash
# 升级到最新版本
migrate -path ./migrations \
    -database "mysql://root:123456@tcp(139.196.242.169:3306)/video_server?charset=utf8mb4&parseTime=True" \
    up
```

#### Step 5: 集成到项目（可选）

**方式1：启动脚本中执行**
```bash
#!/bin/bash
# deploy.sh

# 1. 先执行migration
migrate -path ./migrations -database "$DB_URL" up

# 2. 再启动服务
./bin/api-service
```

**方式2：Go代码中集成**
```go
// cmd/migrate/main.go
package main

import (
    "log"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/mysql"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    m, err := migrate.New(
        "file://migrations",
        "mysql://root:pwd@tcp(localhost:3306)/video_server",
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        log.Fatal(err)
    }

    log.Println("✓ Migrations applied successfully")
}

// 运行：go run cmd/migrate/main.go
```

#### Step 6: 未来添加新字段

```bash
# 1. 创建migration
migrate create -ext sql -dir migrations -seq add_email_to_users

# 2. 编写SQL
echo "ALTER TABLE users ADD COLUMN email VARCHAR(255);" > migrations/000002_add_email_to_users.up.sql
echo "ALTER TABLE users DROP COLUMN email;" > migrations/000002_add_email_to_users.down.sql

# 3. 执行
migrate -path ./migrations -database "..." up

# 4. 提交到Git
git add migrations/000002*
git commit -m "feat: 添加用户邮箱字段"
```

---

## 🚀 最佳实践

### 1. Migration命名规范

```
{序号}_{描述}.{up|down}.sql

✅ 好的命名：
  000001_create_users_table.up.sql
  000002_add_email_to_users.up.sql
  000003_add_index_to_videos.up.sql

❌ 不好的命名：
  update.sql
  fix_bug.sql
  20230315.sql
```

### 2. 每个Migration只做一件事

```sql
-- ✅ 好的做法：单一职责
-- 000002_add_email_to_users.up.sql
ALTER TABLE users ADD COLUMN email VARCHAR(255);

-- 000003_add_index_to_email.up.sql
CREATE INDEX idx_users_email ON users(email);

-- ❌ 不好的做法：混在一起
-- 000002_various_updates.up.sql
ALTER TABLE users ADD COLUMN email VARCHAR(255);
ALTER TABLE videos ADD COLUMN duration INT;
CREATE INDEX idx_comments_created ON comments(create_time);
```

### 3. 永远提供Down脚本

```sql
-- ✅ 必须提供回滚
-- 000002_add_email_to_users.down.sql
ALTER TABLE users DROP COLUMN email;

-- ❌ 不写down脚本：出问题无法回滚
```

### 4. 测试Migration

```bash
# 在本地/测试环境先测试
migrate up    # 升级
migrate down  # 回滚
migrate up    # 再次升级

# 确保可以来回切换无问题
```

### 5. 数据迁移要小心

```sql
-- 如果有数据迁移逻辑，要考虑大表性能
-- ❌ 危险：锁表时间长
UPDATE users SET email = CONCAT(name, '@example.com');

-- ✅ 分批处理
UPDATE users SET email = CONCAT(name, '@example.com')
WHERE id BETWEEN 1 AND 10000;
-- 再执行 10001-20000...
```

---

## 📊 对比总结

| 维度 | 手动执行SQL | Migration管理 |
|------|------------|--------------|
| **版本控制** | ❌ 无记录 | ✅ 有版本号 |
| **可回滚** | ❌ 手动回滚，容易出错 | ✅ 自动回滚 |
| **团队协作** | ❌ 每人手动同步 | ✅ Git拉代码自动同步 |
| **环境一致性** | ❌ 容易不一致 | ✅ 保证一致 |
| **CI/CD集成** | ❌ 难集成 | ✅ 容易集成 |
| **新人加入** | ❌ 需要找SQL手动执行 | ✅ 运行migrate即可 |
| **生产部署** | ❌ 容易遗漏 | ✅ 自动化部署 |

---

## 🎯 核心价值

1. **可追溯**：知道数据库什么时候、被谁、因为什么改了
2. **可回滚**：出问题可以快速回到上一个稳定版本
3. **可重现**：新环境、新成员、新服务器一键同步数据库
4. **可自动化**：CI/CD流程中自动执行，减少人为错误

---

**生成时间**: 2026-03-15
**适用项目**: video-server
