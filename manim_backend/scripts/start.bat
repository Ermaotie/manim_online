@echo off
echo === Manim Backend 启动脚本 ===

REM 检查Go环境
echo 检查Go环境...
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: Go未安装
    pause
    exit /b 1
)

echo Go版本:
go version

REM 下载依赖
echo 下载依赖...
go mod download

REM 运行测试
echo 运行单元测试...
go test ./... -v
if %errorlevel% neq 0 (
    echo 测试失败，停止启动
    pause
    exit /b 1
)

echo 所有测试通过!

REM 构建应用
echo 构建应用...
go build -o manim-backend.exe main.go
if %errorlevel% neq 0 (
    echo 构建失败
    pause
    exit /b 1
)

echo 构建成功!

REM 创建必要的目录
echo 创建必要目录...
if not exist logs mkdir logs
if not exist temp mkdir temp
if not exist videos mkdir videos

REM 启动应用
echo 启动Manim Backend服务...
manim-backend.exe -f etc\manim-backend.yaml

pause