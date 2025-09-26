# Manim Backend API 文档

## 概述

Manim Backend 是一个基于Go语言的后端服务，用于将自然语言描述转换为Manim动画代码并生成视频。系统支持用户认证、视频管理、AI代码生成等功能。

**基础信息**
- **服务器地址**: `http://localhost:8888`
- **API前缀**: `/api`
- **认证方式**: Bearer Token

## 认证

### 用户注册

注册新用户账户。

**请求**
```http
POST /api/auth/register
Content-Type: application/json

{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123"
}
```

**响应**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com"
  }
}
```

### 用户登录

用户登录获取访问令牌。

**请求**
```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}
```

**响应**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com"
  }
}
```

## 用户管理

### 获取用户信息

获取当前登录用户的详细信息。

**请求**
```http
GET /api/user/profile
Authorization: Bearer <token>
```

**响应**
```json
{
  "id": 1,
  "username": "testuser",
  "email": "test@example.com"
}
```

## 视频管理

### 创建视频任务

创建一个新的视频生成任务。系统会异步处理视频渲染。

**请求**
```http
POST /api/videos
Authorization: Bearer <token>
Content-Type: application/json

{
  "prompt": "创建一个演示勾股定理的动画，展示直角三角形三边的关系"
}
```

**响应**
```json
{
  "id": 1,
  "prompt": "创建一个演示勾股定理的动画，展示直角三角形三边的关系",
  "status": "pending",
  "created_at": "2025-01-25 21:30:15",
  "updated_at": "2025-01-25 21:30:15"
}
```

### 获取视频列表

获取当前用户的视频列表，支持分页。

**请求**
```http
GET /api/videos?page=1&page_size=10
Authorization: Bearer <token>
```

**查询参数**
- `page` (可选): 页码，默认1
- `page_size` (可选): 每页数量，默认10，最大100

**响应**
```json
{
  "videos": [
    {
      "id": 1,
      "prompt": "创建一个演示勾股定理的动画，展示直角三角形三边的关系",
      "video_path": "/videos/3/PythagoreanTheorem.mp4",
      "status": "completed",
      "error_msg": "",
      "created_at": "2025-01-25 21:30:15",
      "updated_at": "2025-01-25 21:30:45"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10
}
```

### 获取视频详情

获取指定视频的详细信息。

**请求**
```http
GET /api/videos/detail?id=1
Authorization: Bearer <token>
```

**查询参数**
- `id` (必需): 视频ID

**响应**
```json
{
  "id": 1,
  "prompt": "创建一个演示勾股定理的动画，展示直角三角形三边的关系",
  "manim_code": "from manim import *\n\nclass PythagoreanTheorem(Scene):\n    def construct(self):\n        # 创建直角三角形\n        triangle = Polygon(ORIGIN, RIGHT*3, UP*4, color=BLUE)\n        self.play(Create(triangle))\n        \n        # 添加标签\n        labels = VGroup(\n            Text("a").next_to(triangle.get_vertices()[1], DOWN),\n            Text("b").next_to(triangle.get_vertices()[2], LEFT),\n            Text("c").next_to(triangle.get_center(), RIGHT+UP)\n        )\n        self.play(Write(labels))\n        \n        self.wait(2)",
  "video_path": "/videos/3/PythagoreanTheorem.mp4",
  "status": "completed",
  "error_msg": "",
  "created_at": "2025-01-25 21:30:15",
  "updated_at": "2025-01-25 21:30:45"
}
```

### 删除视频

删除指定的视频记录和文件。

**请求**
```http
DELETE /api/videos?id=1
Authorization: Bearer <token>
```

**查询参数**
- `id` (必需): 视频ID

**响应**
```json
{
  "message": "视频删除成功"
}
```

## AI代码生成

### 生成Manim代码

根据自然语言描述生成Manim动画代码。

**请求**
```http
POST /api/ai/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "prompt": "创建一个圆形从左侧移动到右侧的动画"
}
```

**响应**
```json
{
  "code": "from manim import *\n\nclass CircleAnimation(Scene):\n    def construct(self):\n        circle = Circle(radius=1, color=BLUE)\n        circle.move_to(LEFT*5)\n        \n        self.play(Create(circle))\n        self.play(circle.animate.move_to(RIGHT*5), run_time=3)\n        self.wait(1)",
  "is_valid": true,
  "message": "代码生成成功"
}
```

### 验证Manim代码

验证Manim代码的语法正确性。

**请求**
```http
POST /api/ai/validate
Authorization: Bearer <token>
Content-Type: application/json

{
  "code": "from manim import *\n\nclass TestScene(Scene):\n    def construct(self):\n        circle = Circle()\n        self.play(Create(circle))"
}
```

**响应**
```json
{
  "is_valid": true,
  "message": "代码验证通过"
}
```

### 检查AI API健康状态

检查OpenAI API的连接状态和配置有效性。

**请求**
```http
GET /api/ai/health
Authorization: Bearer <token>
```

**响应**
```json
{
  "status": "success",
  "message": "API连接正常",
  "is_healthy": true
}
```

## 视频文件访问

### 访问生成的视频文件

通过URL直接访问生成的视频文件。

**请求**
```http
GET /videos/{user_id}/{filename}
```

**示例**
```http
GET /videos/3/PythagoreanTheorem.mp4
```

**说明**
- `user_id`: 用户ID
- `filename`: 视频文件名（保持原始文件名，如 `PythagoreanTheorem.mp4`）

## 视频状态说明

| 状态 | 说明 |
|------|------|
| `pending` | 视频任务已创建，等待处理 |
| `processing` | 视频正在渲染中 |
| `completed` | 视频渲染完成，可访问 |
| `failed` | 视频渲染失败 |

## 错误处理

所有API错误都返回统一的错误格式：

```json
{
  "error": "错误描述信息"
}
```

### 常见HTTP状态码

- `200`: 请求成功
- `400`: 请求参数错误
- `401`: 未授权访问
- `403`: 禁止访问
- `404`: 资源不存在
- `500`: 服务器内部错误

## 异步处理流程

1. **创建任务**: 用户提交视频创建请求
2. **队列处理**: 任务进入处理队列
3. **AI代码生成**: 系统将自然语言转换为Manim代码
4. **视频渲染**: 使用Manim渲染视频
5. **文件处理**: 视频文件移动到最终位置，保持原始文件名
6. **状态更新**: 数据库立即更新视频路径和状态
7. **结果返回**: 用户可通过API查询处理结果

## 并发控制

系统支持同时处理多个视频任务，通过信号量机制控制并发数量：
- 默认最大并发数：5个视频渲染任务
- 队列管理：超出并发限制的任务进入等待队列

## 文件命名规则

- **原始文件名**: 保持Manim生成的原始文件名（如 `PythagoreanTheorem.mp4`）
- **存储路径**: `/videos/{user_id}/{original_filename}`
- **不重命名**: 视频生成后不重命名为 `video.mp4`

## 技术特性

- **异步处理**: 视频渲染过程完全异步
- **智能文件查找**: 支持多层嵌套目录结构的视频文件查找
- **错误恢复**: 自动重试和错误处理机制
- **资源管理**: 自动清理过期视频文件和记录
- **实时进度**: 支持视频渲染进度监控

## 使用示例

### Python客户端示例

```python
import requests

# 登录获取token
login_data = {
    "username": "testuser",
    "password": "password123"
}
response = requests.post("http://localhost:8888/api/auth/login", json=login_data)
token = response.json()["token"]

# 创建视频任务
headers = {"Authorization": f"Bearer {token}"}
video_data = {
    "prompt": "创建一个正方形旋转的动画"
}
response = requests.post("http://localhost:8888/api/videos", json=video_data, headers=headers)
video_id = response.json()["id"]

# 查询视频状态
response = requests.get(f"http://localhost:8888/api/videos/detail?id={video_id}", headers=headers)
video_info = response.json()
print(f"视频状态: {video_info['status']}")
```

## 更新日志

### 最新更新
- 视频生成后保持原始文件名，不重命名为 `video.mp4`
- 视频移动完成后立即更新数据库 `video_path` 字段
- 增强视频文件查找逻辑，支持临时目录内嵌套路径
- 优化文件处理效率，使用移动而非复制操作

---

**注意**: 本文档基于当前系统版本，API可能会随版本更新而变化。建议定期查看最新文档。