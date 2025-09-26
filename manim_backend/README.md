# Manim Backend

一个基于Go-zero框架的Manim动画生成服务后端，支持自然语言生成Manim代码并渲染视频。

## 功能特性

- ✅ 用户注册登录认证
- ✅ 自然语言生成Manim代码
- ✅ 异步视频渲染处理
- ✅ 支持多视频并发处理
- ✅ 自动清理过期视频
- ✅ 完整的单元测试覆盖
- ✅ 模块化设计，易于扩展

## 技术栈

- **后端框架**: Go-zero
- **数据库**: MySQL
- **缓存**: Redis
- **AI服务**: OpenAI API
- **动画渲染**: Manim
- **认证**: JWT

## 项目结构

```
manim_backend/
├── etc/                    # 配置文件
│   └── manim-backend.yaml  # 应用配置
├── internal/               # 内部模块
│   ├── config/            # 配置结构体
│   ├── handler/           # HTTP处理器
│   ├── middleware/        # 中间件
│   ├── model/             # 数据模型
│   ├── service/           # 业务逻辑服务
│   ├── svc/               # 服务上下文
│   └── types/             # 请求/响应类型
├── scripts/               # 启动脚本
├── main.go               # 应用入口
└── go.mod               # 依赖管理
```

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 5.7+
- Redis 6.0+
- Python 3.8+ (用于Manim)
- Manim Community Edition

### 安装依赖

1. 安装Go依赖:
```bash
go mod download
```

2. 安装Python依赖:
```bash
pip install manim
```

### 配置

1. 复制配置文件模板:
```bash
cp etc/manim-backend.yaml.example etc/manim-backend.yaml
```

2. 编辑配置文件 `etc/manim-backend.yaml`:
```yaml
Name: manim-backend
Host: 0.0.0.0
Port: 8888

OpenAI:
  APIKey: "your-openai-api-key"
  BaseURL: ""

Manim:
  PythonPath: "python"
  MaxConcurrent: 5
  Timeout: 300

Redis:
  Host: 127.0.0.1
  Port: 6379
  Password: ""
  DB: 0

MySQL:
  Host: 127.0.0.1
  Port: 3306
  User: root
  Password: "your-mysql-password"
  DBName: manim_db
```

3. 设置环境变量:
```bash
export OPENAI_API_KEY="your-openai-api-key"
export MYSQL_PASSWORD="your-mysql-password"
```

### 数据库初始化

应用启动时会自动创建所需的表结构。

### 运行测试

```bash
# 运行所有测试
go test ./... -v

# 运行特定包的测试
go test ./internal/service -v
```

### 启动服务

#### Linux/Mac:
```bash
chmod +x scripts/start.sh
./scripts/start.sh
```

#### Windows:
```cmd
scripts\start.bat
```

#### 手动启动:
```bash
go run main.go -f etc/manim-backend.yaml
```

## API文档

### 认证相关

#### 用户注册
```http
POST /api/auth/register
Content-Type: application/json

{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123"
}
```

#### 用户登录
```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}
```

### 视频相关

#### 创建视频任务
```http
POST /api/videos
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "我的动画",
  "description": "这是一个测试动画",
  "prompt": "创建一个圆形从左侧移动到右侧的动画"
}
```

#### 获取视频列表
```http
GET /api/videos?page=1&page_size=10
Authorization: Bearer <token>
```

#### 获取视频详情
```http
GET /api/videos/detail?id=1
Authorization: Bearer <token>
```

#### 删除视频
```http
DELETE /api/videos?id=1
Authorization: Bearer <token>
```

### 视频文件访问

生成的视频文件可以通过以下URL访问:
```http
GET /videos/{video_id}/animation.mp4
```

## 功能说明

### 自然语言生成Manim代码

系统使用OpenAI API将用户的自然语言描述转换为可执行的Manim代码。生成的代码会经过语法验证确保正确性。

### 异步视频处理

视频渲染过程是异步的：
1. 用户提交请求后立即返回任务ID
2. 系统在后台处理Manim代码生成和视频渲染
3. 用户可以通过轮询API获取处理状态

### 并发控制

系统支持同时处理多个视频任务，通过信号量机制控制并发数量，避免资源耗尽。

### 自动清理

系统会自动清理：
- 超过指定天数的已完成视频记录
- 过期的视频文件
- 避免存储空间无限增长

## 测试

项目包含完整的单元测试，覆盖核心业务逻辑：

- 用户服务测试 (注册、登录、查询)
- 视频服务测试 (创建、查询、更新、删除)
- AI服务测试 (代码验证、格式化)

运行测试确保代码质量:
```bash
go test ./... -v
```

## 部署

### 生产环境配置

1. 使用环境变量管理敏感信息
2. 配置反向代理 (Nginx)
3. 设置数据库连接池
4. 配置日志轮转
5. 设置监控和告警

### Docker部署

项目支持Docker部署，具体配置参考Dockerfile。

## 开发指南

### 添加新的API端点

1. 在 `internal/types` 中添加请求/响应类型
2. 在 `internal/handler` 中添加处理器
3. 在 `internal/service` 中添加业务逻辑
4. 在 `internal/handler/routes.go` 中注册路由
5. 添加相应的单元测试

### 扩展功能

项目采用模块化设计，易于扩展：
- 添加新的AI模型支持
- 支持不同的动画引擎
- 集成第三方存储服务
- 添加用户权限管理

## 故障排除

### 常见问题

1. **Manim执行失败**: 检查Python环境和Manim安装
2. **OpenAI API调用失败**: 检查API密钥和网络连接
3. **数据库连接失败**: 检查数据库配置和网络
4. **视频文件无法访问**: 检查文件权限和路径配置

### 日志查看

应用日志保存在 `logs/` 目录下，包含详细的错误信息和调试信息。

## 贡献

欢迎提交Issue和Pull Request来改进项目。

## 许可证

MIT License

## 联系方式

如有问题请联系开发团队。