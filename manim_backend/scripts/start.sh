#!/bin/bash

echo "=== Manim Backend 启动脚本 ==="

# 检查Go环境
echo "检查Go环境..."
if ! command -v go &> /dev/null; then
    echo "错误: Go未安装"
    exit 1
fi

echo "Go版本: $(go version)"

# 下载依赖
echo "下载依赖..."
go mod download

# 运行测试
echo "运行单元测试..."
if ! go test ./... -v; then
    echo "测试失败，停止启动"
    exit 1
fi

echo "所有测试通过!"

# 构建应用
echo "构建应用..."
if ! go build -o manim-backend main.go; then
    echo "构建失败"
    exit 1
fi

echo "构建成功!"

# 创建必要的目录
echo "创建必要目录..."
mkdir -p logs temp videos

# 启动应用
echo "启动Manim Backend服务..."
./manim-backend -f etc/manim-backend.yaml