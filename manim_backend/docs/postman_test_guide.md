# Postman测试指南

本文档详细说明如何使用Postman测试Manim后端API。

## 环境准备

### 1. 启动后端服务

首先确保后端服务正在运行：

```bash
cd d:\TRAE\manim_online\manim_backend
# 构建项目
go build -o manim-backend.exe main.go
# 启动服务
./manim-backend.exe
```

服务默认运行在 `http://localhost:8888`

### 2. 配置环境变量

确保已设置SiliconFlow API密钥：

```bash
# Windows PowerShell
$env:SILICONFLOW_API_KEY="your-api-key-here"

# 或者添加到系统环境变量
```

## Postman测试步骤

### 步骤1：用户注册

**请求信息：**
- **方法：** POST
- **URL：** `http://localhost:8888/api/auth/register`
- **Headers：** `Content-Type: application/json`

**请求体：**
```json
{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
}
```

**预期响应：**
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

### 步骤2：用户登录

**请求信息：**
- **方法：** POST
- **URL：** `http://localhost:8888/api/auth/login`
- **Headers：** `Content-Type: application/json`

**请求体：**
```json
{
    "username": "testuser",
    "password": "password123"
}
```

**预期响应：**
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

**重要：** 保存返回的token，后续API调用需要用到。

### 步骤3：创建视频生成任务（勾股定理演示）

**请求信息：**
- **方法：** POST
- **URL：** `http://localhost:8888/api/videos`
- **Headers：** 
  - `Content-Type: application/json`
  - `Authorization: Bearer <your-token>`

**请求体（勾股定理演示）：**
```json
{
    "title": "勾股定理演示",
    "description": "演示勾股定理的动画",
    "prompt": "请生成一个演示勾股定理的Manim动画代码。要求：创建一个直角三角形，边长分别为3、4、5；在三个边上分别绘制正方形；展示三个正方形的面积关系：3² + 4² = 5²；添加适当的文字说明；使用流畅的动画效果展示。"
}
```

**预期响应：**
```json
{
    "id": 1,
    "title": "勾股定理演示",
    "description": "演示勾股定理的动画",
    "prompt": "请生成一个演示勾股定理的Manim动画代码...",
    "manim_code": "from manim import *\n\nclass PythagoreanTheorem(Scene):\n    def construct(self):\n        # 创建直角三角形\n        triangle = Polygon([-2, -1, 0], [3, -1, 0], [-2, 2, 0])\n        # ... 完整的Manim代码\n        self.play(Create(triangle))\n        # ... 更多动画效果",
    "status": "pending",
    "created_at": "2024-01-01 10:00:00",
    "updated_at": "2024-01-01 10:00:00"
}
```

### 步骤4：获取视频列表

**请求信息：**
- **方法：** GET
- **URL：** `http://localhost:8888/api/videos?page=1&page_size=10`
- **Headers：** `Authorization: Bearer <your-token>`

**预期响应：**
```json
{
    "videos": [
        {
            "id": 1,
            "title": "勾股定理演示",
            "description": "演示勾股定理的动画",
            "prompt": "请生成一个演示勾股定理的Manim动画代码...",
            "video_path": "/videos/pythagorean_theorem.mp4",
            "status": "completed",
            "error_msg": "",
            "created_at": "2024-01-01 10:00:00",
            "updated_at": "2024-01-01 10:10:00"
        }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10
}
```

### 步骤5：获取视频详情

**请求信息：**
- **方法：** GET
- **URL：** `http://localhost:8888/api/videos/detail?id=1`
- **Headers：** `Authorization: Bearer <your-token>`

**预期响应：**
```json
{
    "id": 1,
    "title": "勾股定理演示",
    "description": "演示勾股定理的动画",
    "prompt": "请生成一个演示勾股定理的Manim动画代码...",
    "manim_code": "from manim import *\n\nclass PythagoreanTheorem(Scene):\n    def construct(self):\n        # ... 完整的代码",
    "video_path": "/videos/pythagorean_theorem.mp4",
    "status": "completed",
    "error_msg": "",
    "created_at": "2024-01-01 10:00:00",
    "updated_at": "2024-01-01 10:10:00"
}
```

### 步骤6：AI API健康检查

**请求信息：**
- **方法：** GET
- **URL：** `http://localhost:8888/api/ai/health`
- **Headers：** `Authorization: Bearer <your-token>`

**预期响应（正常情况）：**
```json
{
    "status": "success",
    "message": "API连接正常",
    "is_healthy": true
}
```

**预期响应（配置错误）：**
```json
{
    "status": "error",
    "message": "配置验证失败",
    "details": [
        "API密钥格式可能不正确，应以'sk-'开头",
        "BaseURL格式不正确，必须以http://或https://开头"
    ],
    "is_healthy": false
}
```

**预期响应（API连接失败）：**
```json
{
    "status": "error",
    "message": "API连接失败: 401 Unauthorized",
    "is_healthy": false
}
```

### 步骤7：AI代码生成（独立API）

**请求信息：**
- **方法：** POST
- **URL：** `http://localhost:8888/api/ai/generate`
- **Headers：** 
  - `Content-Type: application/json`
  - `Authorization: Bearer <your-token>`

**请求体：**
```json
{
    "prompt": "请生成一个简单的圆形动画"
}
```

**预期响应：**
```json
{
    "code": "from manim import *\n\nclass CircleAnimation(Scene):\n    def construct(self):\n        circle = Circle(radius=2, color=BLUE)\n        self.play(Create(circle))\n        self.wait(1)",
    "is_valid": true,
    "message": "代码生成成功"
}
```

### 步骤8：代码验证

**请求信息：**
- **方法：** POST
- **URL：** `http://localhost:8888/api/ai/validate`
- **Headers：** 
  - `Content-Type: application/json`
  - `Authorization: Bearer <your-token>`

**请求体：**
```json
{
    "code": "from manim import *\n\nclass TestScene(Scene):\n    def construct(self):\n        pass"
}
```

**预期响应：**
```json
{
    "code": "from manim import *\n\nclass TestScene(Scene):\n    def construct(self):\n        pass",
    "is_valid": true,
    "message": ""
}
```

### 步骤9：获取用户信息

**请求信息：**
- **方法：** GET
- **URL：** `http://localhost:8888/api/user/profile`
- **Headers：** `Authorization: Bearer <your-token>`

**预期响应：**
```json
{
    "id": 1,
    "username": "testuser",
    "email": "test@example.com"
}
```

## Postman集合配置

### 环境变量设置

在Postman中创建环境变量：

```json
{
    "base_url": "http://localhost:8888",
    "token": ""
}
```

### 测试脚本

在每个请求的Tests标签中添加脚本：

**登录请求的Tests脚本：**
```javascript
// 检查响应状态码
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

// 检查响应体包含token
pm.test("Response has token", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.token).to.be.a('string');
    
    // 设置环境变量
    pm.environment.set("token", jsonData.token);
});
```

**需要认证的请求的Pre-request脚本：**
```javascript
// 设置Authorization头
pm.request.headers.add({
    key: 'Authorization',
    value: 'Bearer ' + pm.environment.get("token")
});
```

## 常见问题排查

### 1. 401 Unauthorized错误
- 检查token是否正确设置
- 检查token是否已过期
- 重新登录获取新token

### 2. 500 Internal Server Error
- 检查SiliconFlow API密钥是否正确配置
- 检查后端服务日志查看详细错误信息
- 确认数据库连接正常

### 3. 视频生成失败
- 检查Manim代码是否有效
- 查看视频记录的error_msg字段
- 确认Manim环境配置正确

### 4. API响应缓慢
- 检查AI服务调用是否超时
- 确认网络连接正常
- 检查SiliconFlow API配额使用情况

## 测试用例示例

### 测试用例1：完整的视频生成流程

1. 用户注册/登录
2. 创建勾股定理演示视频任务
3. 轮询查询视频状态直到完成
4. 下载生成的视频文件

### 测试用例2：错误处理

1. 使用无效token访问受保护API
2. 创建视频时使用空描述
3. 访问不存在的视频ID
4. 删除不存在的视频

### 测试用例3：性能测试

1. 并发创建多个视频任务
2. 测试大文件上传
3. 验证API响应时间

## 注意事项

1. **API密钥安全**：不要在代码或配置文件中硬编码API密钥
2. **测试数据清理**：测试完成后清理测试数据
3. **环境隔离**：使用独立的测试环境
4. **错误日志**：详细记录测试过程中的错误信息

通过以上步骤，你可以使用Postman全面测试Manim后端API的所有功能。