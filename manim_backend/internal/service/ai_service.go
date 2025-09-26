package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"manim-backend/internal/config"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
}

func NewAIService(cfg config.OpenAIConfig) *AIService {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	// 设置SiliconFlow API地址
	if cfg.BaseURL == "" {
		clientConfig.BaseURL = "https://api.siliconflow.cn/v1"
	} else {
		// 确保BaseURL格式正确，移除末尾的斜杠
		baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
		clientConfig.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(clientConfig)
	return &AIService{client: client}
}

// GenerateManimCode 根据自然语言生成Manim代码
func (s *AIService) GenerateManimCode(ctx context.Context, prompt string) (string, error) {
	systemPrompt := `你是一个专业的Manim动画代码生成器。请根据用户的自然语言描述生成高质量的Manim代码，仅返回代码即可。

重要要求：
1. 生成的代码必须是完整的、可执行的Python代码
2. 使用Manim社区版（manim）的语法
3. 代码应该包含一个继承自Scene的类
4. 在construct方法中实现动画逻辑
5. 确保代码简洁高效但功能完整

几何和布局要求：
1. 使用坐标系进行精确定位，避免随意放置对象
2. 对于文字和几何对象，使用合理的相对位置关系
3. 使用VGroup、HGroup等容器管理相关对象
4. 使用UP、DOWN、LEFT、RIGHT等方向常量进行定位
5. 对于复杂的几何关系，使用数学计算确保准确性
6. 文字说明应该与对应的几何对象保持适当的间距

动画要求：
1. 使用合理的动画时长和顺序
2. 避免对象重叠或位置冲突
3. 使用合适的颜色和大小区分不同元素
4. 添加适当的等待时间让观众理解动画内容

代码质量要求：
1. 添加详细的注释说明关键步骤
2. 使用有意义的变量名
3. 保持代码结构清晰，避免过于复杂的嵌套
4. 处理可能的边界情况

示例格式：
from manim import *

class ProfessionalAnimation(Scene):
    def construct(self):
        # 创建主要对象
        circle = Circle(radius=1, color=BLUE)
        circle.move_to(ORIGIN)
        
        # 添加文字说明
        label = Text("圆形", font_size=24)
        label.next_to(circle, DOWN, buff=0.3)
        
        # 动画序列
        self.play(Create(circle))
        self.wait(0.5)
        self.play(Write(label))
        self.wait(1)`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       "deepseek-ai/DeepSeek-V3.1",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2000,
	})

	if err != nil {
		return "", fmt.Errorf("AI服务调用失败: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("AI服务返回空结果")
	}

	code := resp.Choices[0].Message.Content

	// 清理代码，确保格式正确
	code = strings.TrimSpace(code)

	// 提取纯Python代码，移除可能的Markdown代码块格式
	code = s.extractPythonCode(code)

	// 检查是否包含必要的导入
	if !strings.Contains(code, "from manim import") && !strings.Contains(code, "import manim") {
		code = "from manim import *\n\n" + code
	}

	// 再次清理代码，确保格式正确
	code = strings.TrimSpace(code)

	return code, nil
}

// extractPythonCode 从AI响应中提取纯Python代码
func (s *AIService) extractPythonCode(content string) string {
	// 移除前后的空白字符
	content = strings.TrimSpace(content)

	// 检查是否包含Markdown代码块
	if strings.HasPrefix(content, "```") {
		// 移除开头的```python或```
		if strings.HasPrefix(content, "```python") {
			content = strings.TrimPrefix(content, "```python")
		} else {
			content = strings.TrimPrefix(content, "```")
		}

		// 移除结尾的```
		if strings.HasSuffix(content, "```") {
			content = strings.TrimSuffix(content, "```")
		}

		content = strings.TrimSpace(content)
	}

	// 检查是否包含其他可能的包装格式
	lines := strings.Split(content, "\n")
	var codeLines []string

	// 跳过可能的解释性文本行，只保留代码行
	inCodeBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和纯注释行（但不跳过代码中的注释）
		if line == "" || strings.HasPrefix(line, "# ") && !strings.Contains(line, "import") {
			continue
		}

		// 检查是否是代码开始（包含Python关键字）
		if strings.HasPrefix(line, "from ") || strings.HasPrefix(line, "import ") ||
			strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "def ") {
			inCodeBlock = true
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
		}
	}

	if len(codeLines) > 0 {
		content = strings.Join(codeLines, "\n")
	}

	// 修复缩进问题
	content = s.fixIndentation(content)

	// 修复Manim代码的常见问题
	content = s.fixManimCode(content)

	return strings.TrimSpace(content)
}

// fixIndentation 修复Python代码的缩进问题
func (s *AIService) fixIndentation(content string) string {
	lines := strings.Split(content, "\n")
	var fixedLines []string

	indentLevel := 0
	inClass := false
	inMethod := false

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")

		// 跳过空行
		if line == "" {
			fixedLines = append(fixedLines, "")
			continue
		}

		// 检查缩进级别
		if strings.HasPrefix(line, "class ") {
			inClass = true
			inMethod = false
			indentLevel = 0
			fixedLines = append(fixedLines, line)
		} else if strings.HasPrefix(line, "def ") {
			inMethod = true
			indentLevel = 1
			fixedLines = append(fixedLines, strings.Repeat("    ", indentLevel)+strings.TrimSpace(line))
		} else if inMethod {
			// 方法内部的代码需要缩进2级
			if strings.TrimSpace(line) != "" {
				fixedLines = append(fixedLines, strings.Repeat("    ", 2)+strings.TrimSpace(line))
			} else {
				fixedLines = append(fixedLines, "")
			}
		} else if inClass {
			// 类内部的代码需要缩进1级
			if strings.TrimSpace(line) != "" {
				fixedLines = append(fixedLines, strings.Repeat("    ", 1)+strings.TrimSpace(line))
			} else {
				fixedLines = append(fixedLines, "")
			}
		} else {
			// 类外部的代码不需要缩进
			fixedLines = append(fixedLines, strings.TrimSpace(line))
		}
	}

	return strings.Join(fixedLines, "\n")
}

// fixManimCode 修复Manim代码的常见问题
func (s *AIService) fixManimCode(content string) string {
	// 确保包含numpy导入
	if strings.Contains(content, "np.") && !strings.Contains(content, "import numpy") {
		// 在manim导入后添加numpy导入
		if strings.Contains(content, "from manim import") {
			content = strings.Replace(content, "from manim import *", "from manim import *\nimport numpy as np", 1)
		} else if strings.Contains(content, "import manim") {
			content = strings.Replace(content, "import manim", "import manim\nimport numpy as np", 1)
		} else {
			// 如果没有manim导入，在开头添加
			content = "from manim import *\nimport numpy as np\n\n" + content
		}
	}

	// 修复常见的几何和布局问题
	content = s.fixGeometryAndLayout(content)

	// 生成改进建议
	improvementSuggestions := s.generateImprovementSuggestions(content)
	if improvementSuggestions != "" {
		// 使用Python兼容的注释格式，避免Unicode符号
		content = "# 代码改进建议：\n" + improvementSuggestions + "\n\n" + content
	}

	return content
}

// fixGeometryAndLayout 修复几何和布局问题
func (s *AIService) fixGeometryAndLayout(content string) string {
	lines := strings.Split(content, "\n")
	var fixedLines []string

	// 保留原始缩进
	for i, line := range lines {
		// 修复随意的位置设置
		if strings.Contains(line, ".move_to(") && !strings.Contains(line, "ORIGIN") &&
			!strings.Contains(line, "UP") && !strings.Contains(line, "DOWN") &&
			!strings.Contains(line, "LEFT") && !strings.Contains(line, "RIGHT") {
			// 建议使用相对位置而不是绝对坐标
			if i > 0 && (strings.Contains(lines[i-1], "=") && (strings.Contains(lines[i-1], "Circle") ||
				strings.Contains(lines[i-1], "Square") || strings.Contains(lines[i-1], "Rectangle"))) {
				// 在对象创建后添加相对位置设置，保持缩进
				indent := getIndentation(line)
				fixedLines = append(fixedLines, indent+"# 使用相对位置进行布局")
			}
		}

		// 修复文字和对象的间距问题
		if strings.Contains(line, "Text(") && i+1 < len(lines) {
			// 检查下一行是否有位置设置
			nextLine := strings.TrimSpace(lines[i+1])
			if !strings.Contains(nextLine, "next_to") && !strings.Contains(nextLine, "move_to") {
				// 建议添加相对位置，保持缩进
				indent := getIndentation(line)
				fixedLines = append(fixedLines, line)
				fixedLines = append(fixedLines, indent+"# 建议使用next_to设置文字相对于几何对象的位置")
				continue
			}
		}

		// 修复缺少缓冲距离的问题
		if strings.Contains(line, "next_to(") && !strings.Contains(line, "buff=") {
			line = strings.Replace(line, "next_to(", "next_to(", 1)
			if strings.Contains(line, ")") {
				line = strings.Replace(line, ")", ", buff=0.3)", 1)
			}
		}

		fixedLines = append(fixedLines, line)
	}

	// 添加布局建议注释
	if !strings.Contains(content, "VGroup") && !strings.Contains(content, "HGroup") {
		// 在合适的位置添加分组建议，保持缩进
		for i, line := range fixedLines {
			if strings.Contains(line, "def construct(self):") && i+1 < len(fixedLines) {
				// 在construct方法开始后添加布局建议
				indent := getIndentation(line)
				newLines := make([]string, 0, len(fixedLines)+2)
				newLines = append(newLines, fixedLines[:i+1]...)
				newLines = append(newLines, indent+"    # 建议：使用VGroup或HGroup管理相关对象")
				newLines = append(newLines, fixedLines[i+1:]...)
				fixedLines = newLines
				break
			}
		}
	}

	return strings.Join(fixedLines, "\n")
}

// getIndentation 获取行的缩进字符串
func getIndentation(line string) string {
	// 查找第一个非空白字符的位置
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			return line[:i]
		}
	}
	// 如果整行都是空白，返回空字符串
	return ""
}

// generateImprovementSuggestions 生成代码改进建议
func (s *AIService) generateImprovementSuggestions(code string) string {
	var suggestions []string

	// 检查布局相关建议
	if strings.Contains(code, "move_to([") {
		suggestions = append(suggestions, "# - 使用相对位置（如next_to、UP、DOWN等）替代绝对坐标，提高布局稳定性")
	}

	if strings.Contains(code, "next_to(") && !strings.Contains(code, "buff=") {
		suggestions = append(suggestions, "# - 在next_to方法中添加buff参数控制对象间距，例如buff=0.3")
	}

	// 检查分组管理
	if (strings.Contains(code, "Circle") || strings.Contains(code, "Square") ||
		strings.Contains(code, "Rectangle")) && !strings.Contains(code, "VGroup") &&
		!strings.Contains(code, "HGroup") {
		suggestions = append(suggestions, "# - 使用VGroup或HGroup管理相关的几何对象，便于整体操作和动画")
	}

	// 检查文字布局
	lines := strings.Split(code, "\n")
	hasText := false
	hasGeometry := false
	textGeometryRelation := false

	for _, line := range lines {
		if strings.Contains(line, "Text(") {
			hasText = true
		}
		if strings.Contains(line, "Circle(") || strings.Contains(line, "Square(") ||
			strings.Contains(line, "Rectangle(") || strings.Contains(line, "Triangle(") {
			hasGeometry = true
		}
		if strings.Contains(line, "next_to(") && (strings.Contains(line, "Text") ||
			strings.Contains(line, "Circle") || strings.Contains(line, "Square") ||
			strings.Contains(line, "Rectangle")) {
			textGeometryRelation = true
		}
	}

	if hasText && hasGeometry && !textGeometryRelation {
		suggestions = append(suggestions, "# - 为文字和几何对象建立明确的相对位置关系，使用next_to方法")
	}

	// 检查动画序列
	animationCount := strings.Count(code, "self.play(")
	if animationCount > 5 {
		suggestions = append(suggestions, "# - 考虑将复杂的动画序列分解为多个场景或使用更简洁的动画效果")
	}

	// 检查颜色和样式
	if !strings.Contains(code, "set_color") && !strings.Contains(code, "set_fill") &&
		!strings.Contains(code, "set_stroke") {
		suggestions = append(suggestions, "# - 为几何对象添加颜色和样式设置，增强视觉效果")
	}

	if len(suggestions) > 0 {
		return strings.Join(suggestions, "\n")
	}

	return ""
}

// ValidateManimCode 验证Manim代码的语法和布局质量
func (s *AIService) ValidateManimCode(ctx context.Context, code string) (bool, string) {
	// 基本验证：检查是否包含必要的Manim元素
	if !strings.Contains(code, "class") || !strings.Contains(code, "Scene") {
		return false, "代码必须包含Scene类定义"
	}

	if !strings.Contains(code, "def construct") {
		return false, "代码必须包含construct方法"
	}

	// 检查是否有动画操作
	if !strings.Contains(code, "self.play") && !strings.Contains(code, "self.wait") {
		return false, "代码应该包含动画操作（play或wait）"
	}

	// 布局质量检查
	layoutIssues := s.checkLayoutQuality(code)
	if layoutIssues != "" {
		return true, fmt.Sprintf("代码验证通过，但有以下布局建议：%s", layoutIssues)
	}

	return true, "代码验证通过，布局质量良好"
}

// checkLayoutQuality 检查代码的布局质量
func (s *AIService) checkLayoutQuality(code string) string {
	var issues []string

	// 检查是否使用绝对坐标
	if strings.Contains(code, "move_to([") || strings.Contains(code, "set_x(") || strings.Contains(code, "set_y(") {
		issues = append(issues, "避免使用绝对坐标，建议使用相对位置（next_to, UP, DOWN等）")
	}

	// 检查是否缺少缓冲距离
	if strings.Contains(code, "next_to(") && !strings.Contains(code, "buff=") {
		issues = append(issues, "建议在next_to中添加buff参数控制间距")
	}

	// 检查文字和几何对象的相对关系
	lines := strings.Split(code, "\n")
	hasText := false
	hasGeometry := false
	textGeometryRelation := false

	for _, line := range lines {
		if strings.Contains(line, "Text(") {
			hasText = true
		}
		if strings.Contains(line, "Circle(") || strings.Contains(line, "Square(") ||
			strings.Contains(line, "Rectangle(") || strings.Contains(line, "Triangle(") {
			hasGeometry = true
		}
		if strings.Contains(line, "next_to(") && (strings.Contains(line, "Text") ||
			strings.Contains(line, "Circle") || strings.Contains(line, "Square") ||
			strings.Contains(line, "Rectangle")) {
			textGeometryRelation = true
		}
	}

	if hasText && hasGeometry && !textGeometryRelation {
		issues = append(issues, "文字和几何对象之间缺少明确的相对位置关系")
	}

	// 检查是否使用分组管理相关对象
	if (strings.Contains(code, "Circle") || strings.Contains(code, "Square") ||
		strings.Contains(code, "Rectangle")) && !strings.Contains(code, "VGroup") &&
		!strings.Contains(code, "HGroup") {
		issues = append(issues, "建议使用VGroup或HGroup管理相关的几何对象")
	}

	// 检查动画序列的合理性
	animationCount := strings.Count(code, "self.play(")
	if animationCount > 5 {
		issues = append(issues, "动画序列可能过长，建议分解为多个场景或使用更简洁的动画")
	}

	if len(issues) > 0 {
		return strings.Join(issues, "; ")
	}

	return ""
}

// CheckAPIHealth 检测API可用性
func (s *AIService) CheckAPIHealth(ctx context.Context) (bool, string) {
	// 为健康检查创建带超时的上下文（10秒超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 创建一个简单的测试请求来验证API连接
	testMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个测试助手，请回复'OK'表示API连接正常。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请回复OK",
		},
	}

	resp, err := s.client.CreateChatCompletion(timeoutCtx, openai.ChatCompletionRequest{
		Model:       "deepseek-ai/DeepSeek-V3.1",
		Messages:    testMessages,
		Temperature: 0.1,
		MaxTokens:   10,
	})

	if err != nil {
		// 检查是否是超时错误
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return false, "API连接超时（10秒）"
		}
		return false, fmt.Sprintf("API连接失败: %v", err)
	}

	if len(resp.Choices) == 0 {
		return false, "API返回空结果"
	}

	responseContent := strings.TrimSpace(resp.Choices[0].Message.Content)
	if strings.Contains(strings.ToLower(responseContent), "ok") {
		return true, "API连接正常"
	}

	return true, fmt.Sprintf("API响应异常，返回内容: %s", responseContent)
}

// ValidateConfig 验证配置文件中的API配置
func (s *AIService) ValidateConfig(cfg config.OpenAIConfig) (bool, []string) {
	var errors []string

	// 检查API密钥
	if cfg.APIKey == "" {
		errors = append(errors, "API密钥不能为空")
	} else if len(cfg.APIKey) < 10 {
		errors = append(errors, "API密钥格式可能不正确")
	}

	// 检查BaseURL
	if cfg.BaseURL == "" {
		errors = append(errors, "BaseURL为空，将使用默认SiliconFlow地址")
	} else if !strings.HasPrefix(cfg.BaseURL, "http") {
		errors = append(errors, "BaseURL格式不正确，必须以http://或https://开头")
	}

	// 检查模型配置
	if !strings.Contains(cfg.APIKey, "sk-") {
		errors = append(errors, "API密钥格式可能不正确，应以'sk-'开头")
	}

	if len(errors) > 0 {
		return false, errors
	}

	return true, nil
}
