# 数据库设置指南

## 概述
这个文档描述了如何为 Manim 在线平台设置 MySQL 数据库。

## 数据库配置

根据配置文件 `etc/manim-backend.yaml`，数据库配置如下：

```yaml
MySQL:
  Host: 127.0.0.1
  Port: 3308
  User: root
  Password: "123456"
  DBName: manim_db
```

## 初始化步骤

### 1. 创建数据库

使用提供的 SQL 脚本创建数据库和表结构：

```bash
# 进入脚本目录
cd scripts

# 使用 MySQL 命令行执行脚本
mysql -u root -p < init_database.sql

# 或者手动执行
mysql -u root -p
# 然后在 MySQL 提示符下执行：
# source init_database.sql
```

### 2. 验证安装

执行以下 SQL 命令验证数据库是否正确创建：

```sql
USE manim_db;
SHOW TABLES;
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM videos;
```

### 3. 修改数据库配置（可选）

如果需要修改数据库配置，请编辑 `etc/manim-backend.yaml` 文件：

```yaml
MySQL:
  Host: your_host
  Port: your_port
  User: your_user
  Password: "your_password"
  DBName: manim_db
```

## 表结构说明

### users 表
- 存储用户信息
- 字段：id, username, email, password, created_at, updated_at, deleted_at
- 约束：username 和 email 必须唯一

### videos 表
- 存储视频信息
- 字段：id, user_id, prompt, manim_code, video_path, status, error_msg, created_at, updated_at, deleted_at
- 外键：user_id 引用 users(id)，删除用户时级联删除相关视频
- 状态：0=待处理, 1=处理中, 2=已完成, 3=失败

## 测试数据

脚本中包含了测试用户和视频数据，可以根据需要删除或修改这些测试数据。

## 注意事项

1. 确保 MySQL 服务正在运行
2. 确保数据库端口（默认3308）可用
3. 确保有足够的权限创建数据库和表
4. 生产环境中请使用更强的密码

## 故障排除

如果初始化失败，请检查：
- MySQL 服务是否启动
- 端口是否正确
- 用户名和密码是否正确
- 是否有创建数据库的权限