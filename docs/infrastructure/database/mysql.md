# MySQL 部署和配置指南

## 概述

MySQL 8.0 是视频服务项目的主数据库，负责存储用户信息、视频元数据、评论等核心业务数据。

## 部署方式

### 1. Docker 部署 (推荐)

#### 使用 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    container_name: video_server_mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: ${MYSQL_DATABASE}
      MYSQL_USER: ${MYSQL_USER}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./docker/mysql/init.sql:/docker-entrypoint-initdb.d/init.sql
    command: --default-authentication-plugin=mysql_native_password
    networks:
      - video_network

volumes:
  mysql_data:

networks:
  video_network:
    driver: bridge
```

#### 环境变量配置

创建 `.env` 文件：

```bash
MYSQL_ROOT_PASSWORD=Cz05180921.
MYSQL_DATABASE=video_server
MYSQL_USER=video_user
MYSQL_PASSWORD=video_password
```

#### 启动服务

```bash
# 启动MySQL服务
docker-compose up -d mysql

# 查看服务状态
docker-compose ps mysql

# 查看日志
docker-compose logs -f mysql
```

### 2. 独立安装

#### macOS

```bash
# 使用Homebrew安装
brew install mysql

# 启动MySQL服务
brew services start mysql

# 安全配置
mysql_secure_installation
```

#### Ubuntu/Debian

```bash
# 更新包列表
sudo apt update

# 安装MySQL
sudo apt install mysql-server

# 启动服务
sudo systemctl start mysql
sudo systemctl enable mysql

# 安全配置
sudo mysql_secure_installation
```

#### Windows

从 [MySQL官网](https://dev.mysql.com/downloads/mysql/) 下载安装包并按照向导安装。

## 数据库初始化

### 表结构

MySQL启动时会自动执行 `docker/mysql/init.sql` 脚本创建以下表：

1. **users**: 用户表
2. **video_info**: 视频信息表
3. **comments**: 评论表
4. **sessions**: 会话表
5. **video_del_rec**: 视频删除记录表

### 初始化脚本内容

```sql
-- 创建数据库
CREATE DATABASE IF NOT EXISTS video_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE video_server;

-- 用户表
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    login_name VARCHAR(64) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 视频信息表
CREATE TABLE video_info (
    id VARCHAR(64) PRIMARY KEY,
    author_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    play_url VARCHAR(255) NOT NULL,
    cover_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(id)
);

-- 评论表
CREATE TABLE comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    video_id VARCHAR(64) NOT NULL,
    user_id INT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (video_id) REFERENCES video_info(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 会话表
CREATE TABLE sessions (
    session_id VARCHAR(255) PRIMARY KEY,
    user_id INT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 视频删除记录表
CREATE TABLE video_del_rec (
    id INT AUTO_INCREMENT PRIMARY KEY,
    video_id VARCHAR(64) NOT NULL,
    deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 连接配置

### 应用连接参数

```yaml
# config.yaml
database:
  host: "localhost"  # Docker环境下使用 "mysql"
  port: 3306
  user: "video_user"
  password: "video_password"
  name: "video_server"
  charset: "utf8mb4"
```

### 连接字符串

```
# 格式
user:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local

# 示例
video_user:video_password@tcp(localhost:3306)/video_server?charset=utf8mb4&parseTime=True&loc=Local
```

## 性能优化

### 1. 连接池配置

```go
// Go应用中的MySQL连接池配置
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
    panic("failed to connect database")
}

sqlDB, err := db.DB()
if err != nil {
    panic("failed to get sql.DB")
}

// 设置连接池参数
sqlDB.SetMaxIdleConns(10)    // 最大空闲连接数
sqlDB.SetMaxOpenConns(100)   // 最大打开连接数
sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
```

### 2. 索引优化

```sql
-- 为常用查询字段添加索引
ALTER TABLE video_info ADD INDEX idx_author_id (author_id);
ALTER TABLE video_info ADD INDEX idx_created_at (created_at);
ALTER TABLE comments ADD INDEX idx_video_id (video_id);
ALTER TABLE comments ADD INDEX idx_user_id (user_id);
```

### 3. 查询优化

```sql
-- 使用EXPLAIN分析查询性能
EXPLAIN SELECT * FROM video_info WHERE author_id = 1 ORDER BY created_at DESC LIMIT 10;

-- 避免SELECT *
SELECT id, title, play_url FROM video_info WHERE author_id = 1;
```

## 备份与恢复

### 1. 数据备份

```bash
# 逻辑备份
mysqldump -u root -p video_server > backup_$(date +%Y%m%d_%H%M%S).sql

# 物理备份 (Docker环境)
docker exec video_server_mysql mysqldump -u root -p video_server > backup.sql
```

### 2. 数据恢复

```bash
# 恢复数据
mysql -u root -p video_server < backup.sql

# Docker环境恢复
docker exec -i video_server_mysql mysql -u root -p video_server < backup.sql
```

### 3. 自动备份脚本

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/path/to/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/backup_$DATE.sql"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 执行备份
mysqldump -u root -p${MYSQL_ROOT_PASSWORD} video_server > $BACKUP_FILE

# 删除7天前的备份
find $BACKUP_DIR -name "backup_*.sql" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE"
```

## 监控与维护

### 1. 健康检查

```bash
# 检查MySQL服务状态
docker-compose exec mysql mysqladmin ping -h localhost

# 检查数据库连接
mysql -u video_user -pvideo_password -e "SELECT 1;"
```

### 2. 性能监控

```sql
-- 查看当前连接数
SHOW STATUS LIKE 'Threads_connected';

-- 查看慢查询
SHOW VARIABLES LIKE 'slow_query_log';
SHOW VARIABLES LIKE 'long_query_time';

-- 查看表大小
SELECT 
    table_name,
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS 'Size (MB)'
FROM information_schema.tables 
WHERE table_schema = 'video_server'
ORDER BY (data_length + index_length) DESC;
```

### 3. 日志配置

```ini
# my.cnf 配置文件
[mysqld]
# 错误日志
log-error = /var/log/mysql/error.log

# 慢查询日志
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 2

# 通用查询日志
general_log = 1
general_log_file = /var/log/mysql/general.log
```

## 安全配置

### 1. 用户权限管理

```sql
-- 创建应用专用用户
CREATE USER 'video_app'@'%' IDENTIFIED BY 'strong_password';

-- 授予必要权限
GRANT SELECT, INSERT, UPDATE, DELETE ON video_server.* TO 'video_app'@'%';

-- 刷新权限
FLUSH PRIVILEGES;
```

### 2. 网络安全

```yaml
# 限制MySQL只监听内网IP
services:
  mysql:
    # ... 其他配置
    ports:
      - "127.0.0.1:3306:3306"  # 只绑定本地回环地址
```

### 3. SSL/TLS配置

```ini
# my.cnf
[mysqld]
ssl-ca = /path/to/ca.pem
ssl-cert = /path/to/server-cert.pem
ssl-key = /path/to/server-key.pem
```

## 故障排除

### 常见问题及解决方案

1. **连接拒绝**
   ```bash
   # 检查端口是否开放
   netstat -tlnp | grep 3306
   
   # 检查防火墙设置
   sudo ufw status
   ```

2. **权限错误**
   ```sql
   -- 检查用户权限
   SHOW GRANTS FOR 'video_user'@'%';
   
   -- 重新授予权限
   GRANT ALL PRIVILEGES ON video_server.* TO 'video_user'@'%';
   FLUSH PRIVILEGES;
   ```

3. **磁盘空间不足**
   ```bash
   # 检查磁盘使用情况
   df -h
   
   # 清理二进制日志
   PURGE BINARY LOGS BEFORE DATE_SUB(NOW(), INTERVAL 7 DAY);
   ```

4. **性能问题**
   ```sql
   -- 查看慢查询
   SET GLOBAL slow_query_log = 'ON';
   
   -- 分析表
   ANALYZE TABLE users, video_info, comments;
   ```

## 升级指南

### MySQL版本升级

```bash
# 1. 备份数据
mysqldump -u root -p video_server > backup_before_upgrade.sql

# 2. 停止服务
docker-compose stop mysql

# 3. 修改docker-compose.yml中的镜像版本
# image: mysql:8.0 -> image: mysql:8.4

# 4. 启动新版本
docker-compose up -d mysql

# 5. 验证升级
docker-compose exec mysql mysql -V
```

## 参考资源

- [MySQL 8.0官方文档](https://dev.mysql.com/doc/refman/8.0/en/)
- [Docker MySQL镜像文档](https://hub.docker.com/_/mysql)
- [MySQL性能调优指南](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)