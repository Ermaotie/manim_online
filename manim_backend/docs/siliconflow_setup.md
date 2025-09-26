# SiliconFlow API 配置指南

## 概述

Manim 在线平台现在支持使用 SiliconFlow API 来生成 Manim 代码。SiliconFlow 提供了与 OpenAI API 兼容的接口，可以无缝替换 OpenAI 服务。

## 获取 API Key

1. 访问 [SiliconFlow 官网](https://www.siliconflow.cn/)
2. 注册账号并登录
3. 进入控制台，获取 API Key
4. 确保账户有足够的额度

## 环境变量配置

将 SiliconFlow API Key 设置为环境变量：

### Windows (PowerShell)
```powershell
$env:SILICONFLOW_API_KEY = "your_siliconflow_api_key_here"
```

### Linux/macOS (Bash)
```bash
export SILICONFLOW_API_KEY="your_siliconflow_api_key_here"
```

### 永久设置 (Windows)
1. 右键点击"此电脑" → "属性"
2. 点击"高级系统设置"
3. 点击"环境变量"
4. 在"用户变量"或"系统变量"中新建变量：
   - 变量名：`SILICONFLOW_API_KEY`
   - 变量值：你的 API Key

## 配置文件说明

项目配置文件 `etc/manim-backend.yaml` 已更新为使用 SiliconFlow API：

```yaml
OpenAI:
  APIKey: ${SILICONFLOW_API_KEY}
  BaseURL: "https://api.siliconflow.cn/v1"
```

## 支持的模型

SiliconFlow 支持多种模型，包括：
- DeepSeek 系列模型
- Qwen 系列模型
- 以及其他兼容 OpenAI API 的模型

## 模型选择

在 `ai_service.go` 中，当前使用的是 `openai.GPT3Dot5Turbo` 模型标识符。由于 SiliconFlow API 兼容 OpenAI，这个标识符会被映射到 SiliconFlow 平台上对应的模型。

如果需要指定特定的 SiliconFlow 模型，可以修改：

```go
resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
    Model:       "deepseek-chat",  // 使用 DeepSeek 模型
    Messages:    messages,
    Temperature: 0.7,
    MaxTokens:   2000,
})
```

## 测试配置

1. 设置环境变量
2. 启动应用：
   ```bash
   cd d:\TRAE\manim_online\manim_backend
   go run main.go
   ```

3. 测试 AI 代码生成功能

## 故障排除

### 常见问题

1. **API Key 错误**
   - 检查 API Key 是否正确
   - 确保账户有足够额度

2. **网络连接问题**
   - 检查是否能访问 `https://api.siliconflow.cn`
   - 检查防火墙设置

3. **模型不可用**
   - 检查模型标识符是否正确
   - 查看 SiliconFlow 平台支持的模型列表

### 调试信息

如果遇到问题，可以：

1. 检查应用日志
2. 验证环境变量是否设置正确：
   ```bash
   echo $env:SILICONFLOW_API_KEY  # Windows PowerShell
   echo $SILICONFLOW_API_KEY      # Linux/macOS
   ```

## 从 OpenAI 迁移

如果你之前使用 OpenAI API，迁移到 SiliconFlow 非常简单：

1. 将环境变量从 `OPENAI_API_KEY` 改为 `SILICONFLOW_API_KEY`
2. 无需修改代码，因为 API 接口完全兼容
3. 可能需要在 SiliconFlow 平台选择适合的模型

## 成本对比

SiliconFlow 通常提供更具竞争力的价格，具体费用请参考 SiliconFlow 官方定价。