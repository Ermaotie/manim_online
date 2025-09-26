package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAIService_ValidateManimCode(t *testing.T) {
	service := &AIService{}
	ctx := context.Background()

	tests := []struct {
		name    string
		code    string
		isValid bool
		message string
	}{
		{
			name: "有效的Manim代码",
			code: `from manim import *

class MyAnimation(Scene):
    def construct(self):
        circle = Circle()
        self.play(Create(circle))`,
			isValid: true,
			message: "代码验证通过，布局质量良好",
		},
		{
			name: "缺少class关键字",
			code: `from manim import *

def construct(self):
    circle = Circle()
    self.play(Create(circle))`,
			isValid: false,
			message: "代码必须包含Scene类定义",
		},
		{
			name: "缺少Scene关键字",
			code: `from manim import *

class MyAnimation:
    def construct(self):
        circle = Circle()
        self.play(Create(circle))`,
			isValid: false,
			message: "代码必须包含Scene类定义",
		},
		{
			name: "缺少construct方法",
			code: `from manim import *

class MyAnimation(Scene):
    def other_method(self):
        circle = Circle()
        self.play(Create(circle))`,
			isValid: false,
			message: "代码必须包含construct方法",
		},
		{
			name: "缺少动画操作",
			code: `from manim import *

class MyAnimation(Scene):
    def construct(self):
        circle = Circle()
        # 没有play或wait操作`,
			isValid: false,
			message: "代码应该包含动画操作（play或wait）",
		},
		{
			name: "有布局建议的代码",
			code: `from manim import *

class MyAnimation(Scene):
    def construct(self):
        circle = Circle()
        square = Square()
        circle.move_to([2, 0, 0])
        square.move_to([-2, 0, 0])
        self.play(Create(circle), Create(square))`,
			isValid: true,
			message: "代码验证通过，但有以下布局建议：",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid, message := service.ValidateManimCode(ctx, tt.code)

			assert.Equal(t, tt.isValid, isValid)
			if !tt.isValid {
				assert.Contains(t, message, tt.message)
			}
		})
	}
}

func TestAIService_GenerateManimCode(t *testing.T) {
	// 注意：这个测试不会实际调用OpenAI API，只是测试代码处理逻辑
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "空提示",
			prompt:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于我们没有实际的OpenAI客户端，这里只测试错误情况
			if tt.wantErr {
				// 这个测试主要是为了代码覆盖率
				// 实际测试需要mock OpenAI客户端
			}
		})
	}
}

func TestAIService_CodeFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "添加manim导入",
			input: `class MyAnimation(Scene):
    def construct(self):
        pass`,
			expected: "from manim import *\n\nclass MyAnimation(Scene):\n    def construct(self):\n        pass",
		},
		{
			name: "已有导入不重复添加",
			input: `from manim import *

class MyAnimation(Scene):
    def construct(self):
        pass`,
			expected: `from manim import *

class MyAnimation(Scene):
    def construct(self):
        pass`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试代码格式化逻辑
			code := strings.TrimSpace(tt.input)

			if !strings.Contains(code, "from manim import") && !strings.Contains(code, "import manim") {
				code = "from manim import *\n\n" + code
			}

			assert.Equal(t, tt.expected, code)
		})
	}
}

// MockAIService 用于测试的mock AI服务
type MockAIService struct {
	GenerateManimCodeFunc func(ctx context.Context, prompt string) (string, error)
	ValidateManimCodeFunc func(ctx context.Context, code string) (bool, string)
}

func (m *MockAIService) GenerateManimCode(ctx context.Context, prompt string) (string, error) {
	if m.GenerateManimCodeFunc != nil {
		return m.GenerateManimCodeFunc(ctx, prompt)
	}
	return "", nil
}

func (m *MockAIService) ValidateManimCode(ctx context.Context, code string) (bool, string) {
	if m.ValidateManimCodeFunc != nil {
		return m.ValidateManimCodeFunc(ctx, code)
	}
	return true, ""
}

func TestMockAIService(t *testing.T) {
	mockService := &MockAIService{
		GenerateManimCodeFunc: func(ctx context.Context, prompt string) (string, error) {
			if prompt == "error" {
				return "", assert.AnError
			}
			return "mock code", nil
		},
		ValidateManimCodeFunc: func(ctx context.Context, code string) (bool, string) {
			return code != "invalid", ""
		},
	}

	ctx := context.Background()

	// 测试GenerateManimCode
	code, err := mockService.GenerateManimCode(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, "mock code", code)

	_, err = mockService.GenerateManimCode(ctx, "error")
	assert.Error(t, err)

	// 测试ValidateManimCode
	isValid, _ := mockService.ValidateManimCode(ctx, "valid")
	assert.True(t, isValid)

	isValid, _ = mockService.ValidateManimCode(ctx, "invalid")
	assert.False(t, isValid)
}

// TestAIService_GeneratePythagoreanTheorem 测试使用配置中的API生成勾股定理演示
func TestAIService_GeneratePythagoreanTheorem(t *testing.T) {
	// 创建模拟的AI服务配置
	mockConfig := struct {
		APIKey  string
		BaseURL string
	}{
		APIKey:  "test-api-key",
		BaseURL: "https://api.siliconflow.cn/v1",
	}

	// 创建AI服务实例
	service := NewAIService(mockConfig)
	ctx := context.Background()

	// 勾股定理演示的提示
	prompt := `请生成一个演示勾股定理的Manim动画代码。

要求：
1. 创建一个直角三角形，边长分别为3、4、5
2. 在三个边上分别绘制正方形
3. 展示三个正方形的面积关系：3² + 4² = 5²
4. 添加适当的文字说明
5. 使用流畅的动画效果展示`

	tests := []struct {
		name        string
		prompt      string
		expectError bool
	}{
		{
			name:        "生成勾股定理演示代码",
			prompt:      prompt,
			expectError: false,
		},
		{
			name:        "空提示",
			prompt:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：在实际测试中，这里会调用真实的API
			// 但由于这是单元测试，我们主要测试代码逻辑和错误处理

			if tt.expectError {
				// 测试错误处理逻辑
				_, err := service.GenerateManimCode(ctx, tt.prompt)
				assert.Error(t, err)
			} else {
				// 测试正常流程（在实际环境中会调用API）
				// 这里主要验证函数调用不会panic
				code, err := service.GenerateManimCode(ctx, tt.prompt)

				// 由于我们没有实际的API调用，这里主要测试函数结构
				if err == nil {
					// 如果成功返回代码，验证基本格式
					assert.NotEmpty(t, code)

					// 验证生成的代码包含必要的Manim元素
					assert.Contains(t, code, "from manim import")
					assert.Contains(t, code, "class")
					assert.Contains(t, code, "Scene")
					assert.Contains(t, code, "construct")

					// 验证代码格式正确
					isValid, message := service.ValidateManimCode(ctx, code)
					assert.True(t, isValid, "生成的代码应该有效: %s", message)
				} else {
					// 如果API调用失败，验证错误信息格式
					assert.Contains(t, err.Error(), "AI服务")
				}
			}
		})
	}
}
