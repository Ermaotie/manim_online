# Manim Backend 依赖检查脚本
Write-Host "=== Manim Backend 依赖检查 ===" -ForegroundColor Green

# 检查Go环境
Write-Host "检查Go环境..." -ForegroundColor Yellow
$goVersion = go version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: Go未安装" -ForegroundColor Red
    exit 1
}
Write-Host "Go版本: $goVersion" -ForegroundColor Green

# 检查Python环境
Write-Host "检查Python环境..." -ForegroundColor Yellow
$pythonVersion = python --version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "警告: Python未安装或未在PATH中" -ForegroundColor Yellow
} else {
    Write-Host "Python版本: $pythonVersion" -ForegroundColor Green
}

# 检查Manim安装
Write-Host "检查Manim安装..." -ForegroundColor Yellow
$manimVersion = python -c "import manim; print(manim.__version__)" 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "警告: Manim未安装" -ForegroundColor Yellow
} else {
    Write-Host "Manim版本: $manimVersion" -ForegroundColor Green
}

# 检查项目结构
Write-Host "检查项目结构..." -ForegroundColor Yellow
$requiredFiles = @(
    "main.go",
    "go.mod",
    "etc/manim-backend.yaml",
    "internal/config/config.go",
    "internal/handler/routes.go",
    "internal/service/user_service.go",
    "internal/service/video_service.go",
    "internal/service/ai_service.go",
    "internal/service/manim_service.go"
)

$allFilesExist = $true
foreach ($file in $requiredFiles) {
    if (Test-Path $file) {
        Write-Host "✓ $file" -ForegroundColor Green
    } else {
        Write-Host "✗ $file (缺失)" -ForegroundColor Red
        $allFilesExist = $false
    }
}

if ($allFilesExist) {
    Write-Host "项目结构完整" -ForegroundColor Green
} else {
    Write-Host "项目结构不完整" -ForegroundColor Red
}

# 检查配置文件
Write-Host "检查配置文件..." -ForegroundColor Yellow
if (Test-Path "etc/manim-backend.yaml") {
    $configContent = Get-Content "etc/manim-backend.yaml" -Raw
    if ($configContent -match "OPENAI_API_KEY" -or $configContent -match "your-openai-api-key") {
        Write-Host "警告: 需要配置OpenAI API密钥" -ForegroundColor Yellow
    }
    if ($configContent -match "MYSQL_PASSWORD" -or $configContent -match "your-mysql-password") {
        Write-Host "警告: 需要配置MySQL密码" -ForegroundColor Yellow
    }
}

Write-Host "=== 依赖检查完成 ===" -ForegroundColor Green