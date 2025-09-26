package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"manim-backend/internal/config"
	"manim-backend/internal/model"

	"github.com/google/uuid"
)

type ManimService struct {
	cfg          config.ManimConfig
	videoService *VideoService
	semaphore    chan struct{}
	mu           sync.Mutex
}

func NewManimService(cfg config.ManimConfig, videoService *VideoService) *ManimService {
	semaphore := make(chan struct{}, cfg.MaxConcurrent)

	return &ManimService{
		cfg:          cfg,
		videoService: videoService,
		semaphore:    semaphore,
	}
}

// SetVideoService 设置VideoService依赖
func (s *ManimService) SetVideoService(videoService *VideoService) {
	s.videoService = videoService
}

// GenerateVideo 生成视频
func (s *ManimService) GenerateVideo(ctx context.Context, videoID uint, manimCode string) error {
	// 获取信号量，控制并发数量
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	// 更新视频状态为处理中
	err := s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusProcessing, manimCode, "", "")
	if err != nil {
		return err
	}

	// 获取视频信息以获取用户ID
	video, err := s.videoService.GetVideoByID(ctx, videoID)
	if err != nil {
		s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("获取视频信息失败: %v", err))
		return err
	}

	// 创建临时工作目录
	tempDir := filepath.Join("temp", uuid.New().String())
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("创建临时目录失败: %v", err))
		return err
	}
	// defer os.RemoveAll(tempDir) // 注释掉删除临时目录的代码，保留中间文件用于调试

	// 写入Manim代码文件
	codeFile := filepath.Join(tempDir, "animation.py")
	err = os.WriteFile(codeFile, []byte(manimCode), 0644)
	if err != nil {
		s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("写入代码文件失败: %v", err))
		return err
	}

	// 执行Manim命令
	// 使用项目根目录下的videos目录作为输出目录，按用户ID分类
	outputDir := filepath.Join("videos", fmt.Sprintf("%d", video.UserID))
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("创建输出目录失败: %v", err))
		return err
	}

	// 执行Manim渲染命令
	videoPath, err := s.executeManimCommand(ctx, videoID, manimCode, codeFile, outputDir, tempDir)
	if err != nil {
		return err // 错误信息已在executeManimCommand中更新
	}

	// 提取原始文件名（不重命名）
	originalFileName := filepath.Base(videoPath)
	// 将视频文件移动到最终目录，保持原始文件名
	finalVideoPath := filepath.Join("videos", fmt.Sprintf("%d", video.UserID), originalFileName)
	err = s.moveVideoToFinalLocation(videoPath, finalVideoPath, videoID, manimCode)
	if err != nil {
		return err // 错误信息已在moveVideoToFinalLocation中更新
	}

	// 立即更新数据库中的video_path字段
	err = s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusCompleted, manimCode, finalVideoPath, "")
	if err != nil {
		return err
	}

	return nil
}

// executeManimCommand 执行Manim渲染命令
func (s *ManimService) executeManimCommand(ctx context.Context, videoID uint, manimCode, codeFile, outputDir, tempDir string) (string, error) {
	// 根据Manim文档和测试，正确的命令格式是：manim render [OPTIONS] FILE [SCENE_NAMES]
	// 使用 -qm 参数表示中等质量，--media_dir 指定媒体目录
	// 注意：codeFile需要使用绝对路径，因为cmd.Dir设置为tempDir
	absCodeFile, _ := filepath.Abs(codeFile)
	cmd := exec.CommandContext(ctx, s.cfg.PythonPath, "-m", "manim", "render",
		"-qm",                    // 中等质量
		"--media_dir", outputDir, // 媒体目录
		absCodeFile) // Python脚本文件（绝对路径）
	cmd.Dir = tempDir

	// 设置环境变量，确保Manim能找到依赖
	cmd.Env = append(os.Environ(),
		"PYTHONPATH=", // 清空PYTHONPATH避免冲突
	)

	// 使用实时输出监控来获取进度信息
	output, err := s.executeManimWithProgressMonitoring(ctx, cmd, videoID, manimCode)
	if err != nil {
		return "", err
	}

	// 智能等待视频文件完全生成
	videoPath, err := s.waitForVideoFileGeneration(codeFile, outputDir, output)
	if err != nil {
		s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", err.Error())
		return "", err
	}

	return videoPath, nil
}

// executeManimWithProgressMonitoring 执行Manim命令并监控进度
func (s *ManimService) executeManimWithProgressMonitoring(ctx context.Context, cmd *exec.Cmd, videoID uint, manimCode string) ([]byte, error) {
	// 创建管道来捕获实时输出
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建标准输出管道失败: %v", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("创建标准错误管道失败: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动Manim命令失败: %v", err)
	}

	// 创建缓冲区来收集输出
	var outputBuffer strings.Builder

	// 创建进度监控通道
	progressChan := make(chan string, 100)
	doneChan := make(chan bool)

	// 启动输出监控goroutine
	go s.monitorManimOutput(stdoutPipe, stderrPipe, &outputBuffer, progressChan, doneChan, videoID, manimCode)

	// 等待命令完成
	err = cmd.Wait()
	doneChan <- true

	// 检查Manim是否成功执行
	// Windows系统Manim有时会返回STATUS_CONTROL_C_EXIT (0xc000013a)但实际执行成功
	// 需要检查输出内容来判断是否真正失败
	output := []byte(outputBuffer.String())

	if err != nil {
		// 检查是否是Windows特定的控制台退出错误
		if _, ok := err.(*exec.ExitError); ok {
			// Windows系统: 0xc000013a = STATUS_CONTROL_C_EXIT
			// 检查输出中是否包含成功信息
			outputStr := string(output)
			if strings.Contains(outputStr, "File ready at") ||
				strings.Contains(outputStr, "Rendered") ||
				strings.Contains(outputStr, "Played") {
				// Manim实际执行成功，只是退出码异常
				// 继续执行视频文件查找逻辑
			} else {
				// 真正的执行失败
				s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("Manim执行失败: %v, 输出: %s", err, output))
				return output, fmt.Errorf("Manim执行失败: %v, 输出: %s", err, output)
			}
		} else {
			// 其他类型的错误
			s.videoService.UpdateVideoStatus(ctx, videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("Manim执行失败: %v, 输出: %s", err, output))
			return output, fmt.Errorf("Manim执行失败: %v, 输出: %s", err, output)
		}
	}

	return output, nil
}

// monitorManimOutput 监控Manim输出并检测进度
func (s *ManimService) monitorManimOutput(stdoutPipe, stderrPipe io.ReadCloser, outputBuffer *strings.Builder, progressChan chan<- string, doneChan <-chan bool, videoID uint, manimCode string) {
	// 创建扫描器来读取输出
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	stderrScanner := bufio.NewScanner(stderrPipe)

	// 启动goroutine来读取标准输出
	go func() {
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			outputBuffer.WriteString(line + "\n")

			// 检测进度信息
			if progress := s.detectManimProgress(line); progress != "" {
				select {
				case progressChan <- progress:
				default:
					// 通道已满，跳过
				}
			}

			// 检测完成标志
			if s.isManimComplete(line) {
				// 可以在这里更新状态为"渲染完成"
			}
		}
	}()

	// 启动goroutine来读取标准错误
	go func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			outputBuffer.WriteString(line + "\n")

			// 错误信息也可以包含进度信息
			if progress := s.detectManimProgress(line); progress != "" {
				select {
				case progressChan <- progress:
				default:
					// 通道已满，跳过
				}
			}
		}
	}()

	// 等待完成信号
	<-doneChan
}

// detectManimProgress 从Manim输出中检测进度信息
func (s *ManimService) detectManimProgress(line string) string {
	// Manim进度检测模式
	progressPatterns := []struct {
		pattern string
		message string
	}{
		{"Rendering animation", "开始渲染动画"},
		{"Rendered", "动画渲染完成"},
		{"Played", "动画播放完成"},
		{"Writing to", "正在写入文件"},
		{"Animation duration", "动画时长"},
		{"File ready at", "文件准备就绪"},
		{"Scene.*rendered", "场景渲染完成"},
		{"Animation.*completed", "动画完成"},
	}

	for _, p := range progressPatterns {
		if strings.Contains(line, p.pattern) {
			return fmt.Sprintf("%s: %s", p.message, line)
		}
	}

	return ""
}

// isManimComplete 检测Manim是否完成渲染
func (s *ManimService) isManimComplete(line string) bool {
	completionPatterns := []string{
		"File ready at",
		"Rendered",
		"Played",
		"Animation completed",
		"Scene.*rendered",
	}

	for _, pattern := range completionPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}

// waitForVideoFileGeneration 智能等待视频文件生成
func (s *ManimService) waitForVideoFileGeneration(codeFile, outputDir string, manimOutput []byte) (string, error) {
	// 首先检查Manim输出中是否包含完成标志
	outputStr := string(manimOutput)
	manimCompleted := strings.Contains(outputStr, "File ready at") ||
		strings.Contains(outputStr, "Rendered") ||
		strings.Contains(outputStr, "Played")

	// 如果Manim已经完成，额外等待1秒确保文件完全写入
	if manimCompleted {
		time.Sleep(1 * time.Second)
	}

	// 提取Manim代码中的类名
	className := s.extractClassNameFromCode(codeFile)

	// 如果成功提取到类名，使用类名文件监控策略
	if className != "" {
		// 生成预期的视频文件路径模式
		expectedVideoPaths := s.generateExpectedVideoPaths(outputDir, codeFile, className)

		// 使用类名文件监控策略
		videoPath := s.waitForClassNameVideoFile(expectedVideoPaths, 30*time.Second)
		if videoPath != "" {
			return videoPath, nil
		}
	}

	// 如果类名监控失败，回退到原有的多策略查找
	return s.fallbackVideoSearch(codeFile, outputDir, manimOutput)
}

// extractClassNameFromCode 从Manim代码中提取类名
func (s *ManimService) extractClassNameFromCode(codeFile string) string {
	codeContent, err := os.ReadFile(codeFile)
	if err != nil {
		return ""
	}

	// 查找class定义
	lines := strings.Split(string(codeContent), "\n")
	for _, line := range lines {
		if strings.Contains(line, "class ") && strings.Contains(line, "(Scene):") {
			// 提取类名
			parts := strings.Split(strings.TrimSpace(line), " ")
			if len(parts) >= 2 {
				className := strings.TrimSuffix(parts[1], "(Scene):")
				className = strings.TrimSpace(className)
				return className
			}
		}
	}

	return ""
}

// generateExpectedVideoPaths 生成预期的视频文件路径
func (s *ManimService) generateExpectedVideoPaths(outputDir, codeFile, className string) []string {
	fileName := strings.TrimSuffix(filepath.Base(codeFile), ".py")

	// 获取临时目录的父目录（tempDir的父目录）
	tempDir := filepath.Dir(codeFile)

	// 生成多种可能的路径模式，包括临时目录内的路径
	paths := []string{
		// 标准输出目录路径
		filepath.Join(outputDir, "videos", fileName, "720p30", className+".mp4"),
		filepath.Join(outputDir, "videos", fileName, "1080p60", className+".mp4"),
		filepath.Join(outputDir, "videos", fileName, "480p30", className+".mp4"),
		filepath.Join(outputDir, "videos", "animation", "720p30", className+".mp4"),
		filepath.Join(outputDir, "videos", "animation", "1080p60", className+".mp4"),
		filepath.Join(outputDir, "videos", "animation", "480p30", className+".mp4"),
		filepath.Join(outputDir, "videos", "720p30", className+".mp4"),
		filepath.Join(outputDir, "videos", "1080p60", className+".mp4"),
		filepath.Join(outputDir, "videos", "480p30", className+".mp4"),
		
		// 添加缺失的路径模式：包含"-"目录的路径（默认场景）
		filepath.Join(outputDir, "videos", "-", "720p30", className+".mp4"),
		filepath.Join(outputDir, "videos", "-", "1080p60", className+".mp4"),
		filepath.Join(outputDir, "videos", "-", "480p30", className+".mp4"),

		// 临时目录内的路径（根据实际观察的路径结构）
		filepath.Join(tempDir, "videos", fileName, "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", fileName, "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", fileName, "480p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "animation", "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "animation", "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "animation", "480p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "480p30", className+".mp4"),
		
		// 添加缺失的路径模式：包含"-"目录的路径（默认场景）
		filepath.Join(tempDir, "videos", "-", "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "-", "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "-", "480p30", className+".mp4"),

		// 临时目录内嵌套的videos路径（根据实际文件路径）
		filepath.Join(tempDir, "videos", "3", "videos", fileName, "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", fileName, "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", fileName, "480p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "animation", "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "animation", "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "animation", "480p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "480p30", className+".mp4"),
		
		// 添加缺失的路径模式：包含"-"目录的路径（默认场景）
		filepath.Join(tempDir, "videos", "3", "videos", "-", "720p30", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "-", "1080p60", className+".mp4"),
		filepath.Join(tempDir, "videos", "3", "videos", "-", "480p30", className+".mp4"),
	}

	return paths
}

// waitForClassNameVideoFile 等待类名视频文件生成
func (s *ManimService) waitForClassNameVideoFile(expectedPaths []string, maxWaitTime time.Duration) string {
	checkInterval := 500 * time.Millisecond
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		// 检查所有预期的路径
		for _, path := range expectedPaths {
			if _, err := os.Stat(path); err == nil {
				// 文件存在，等待文件稳定
				if s.waitForFileStable(path, 5*time.Second) {
					return path
				}
			}
		}

		// 等待一段时间后再次检查
		time.Sleep(checkInterval)
	}

	return ""
}

// fallbackVideoSearch 回退视频搜索策略
func (s *ManimService) fallbackVideoSearch(codeFile, outputDir string, manimOutput []byte) (string, error) {
	// 首先尝试从Manim输出中提取文件路径
	videoPath := s.extractVideoPathFromOutput(manimOutput)
	if videoPath != "" {
		// 如果从输出中找到了路径，等待文件稳定
		if s.waitForFileStable(videoPath, 30*time.Second) {
			return videoPath, nil
		}
	}

	// 如果从输出中没找到路径，或者文件不稳定，尝试其他查找方法
	fileName := strings.TrimSuffix(filepath.Base(codeFile), ".py")

	// 获取临时目录的父目录
	tempDir := filepath.Dir(codeFile)

	// 等待最多30秒，每500毫秒检查一次
	maxWaitTime := 30 * time.Second
	checkInterval := 500 * time.Millisecond
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		// 尝试各种查找方法，包括临时目录内的搜索
		videoPath := s.findVideoByClassName(codeFile, outputDir, fileName)
		if videoPath != "" && s.waitForFileStable(videoPath, 5*time.Second) {
			return videoPath, nil
		}

		videoPath = s.findVideoByPatterns(outputDir, fileName)
		if videoPath != "" && s.waitForFileStable(videoPath, 5*time.Second) {
			return videoPath, nil
		}

		// 在临时目录内也进行搜索
		videoPath = s.findVideoByPatterns(tempDir, fileName)
		if videoPath != "" && s.waitForFileStable(videoPath, 5*time.Second) {
			return videoPath, nil
		}

		videoPath = s.searchVideoRecursively(outputDir)
		if videoPath != "" && s.waitForFileStable(videoPath, 5*time.Second) {
			return videoPath, nil
		}

		// 在临时目录内也进行递归搜索
		videoPath = s.searchVideoRecursively(tempDir)
		if videoPath != "" && s.waitForFileStable(videoPath, 5*time.Second) {
			return videoPath, nil
		}

		// 等待一段时间后再次检查
		time.Sleep(checkInterval)
	}

	return "", fmt.Errorf("等待视频文件生成超时，Manim输出: %s", manimOutput)
}

// waitForFileStable 等待文件稳定（文件大小不再变化）
func (s *ManimService) waitForFileStable(filePath string, maxWaitTime time.Duration) bool {
	if _, err := os.Stat(filePath); err != nil {
		return false // 文件不存在
	}

	startTime := time.Now()
	checkInterval := 200 * time.Millisecond

	var lastSize int64 = -1
	var stableCount int

	for time.Since(startTime) < maxWaitTime {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return false // 文件访问错误
		}

		currentSize := fileInfo.Size()

		if currentSize == lastSize {
			stableCount++
			// 如果文件大小连续3次检查都相同，认为文件已稳定
			if stableCount >= 3 {
				return true
			}
		} else {
			stableCount = 0
			lastSize = currentSize
		}

		time.Sleep(checkInterval)
	}

	return false // 超时
}

// extractVideoPathFromOutput 从Manim输出中提取视频文件路径
func (s *ManimService) extractVideoPathFromOutput(manimOutput []byte) string {
	outputStr := string(manimOutput)

	// 查找"File ready at"后面的文件路径
	if strings.Contains(outputStr, "File ready at") {
		lines := strings.Split(outputStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "File ready at") {
				// 查找下一行中的文件路径
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					// 检查是否是有效的文件路径
					if strings.HasSuffix(nextLine, ".mp4") && !strings.Contains(nextLine, "partial_movie_files") {
						return nextLine
					}
				}
			}
		}
	}

	return ""
}

// findVideoByClassName 通过类名查找视频文件
func (s *ManimService) findVideoByClassName(codeFile, outputDir, fileName string) string {
	// 读取Manim代码文件，提取类名
	codeContent, err := os.ReadFile(codeFile)
	if err != nil {
		return ""
	}

	// 查找class定义
	lines := strings.Split(string(codeContent), "\n")
	for _, line := range lines {
		if strings.Contains(line, "class ") && strings.Contains(line, "(Scene):") {
			// 提取类名
			parts := strings.Split(strings.TrimSpace(line), " ")
			if len(parts) >= 2 {
				className := strings.TrimSuffix(parts[1], "(Scene):")
				className = strings.TrimSpace(className)

				// 尝试查找该类名对应的视频文件（根据实际测试的路径格式）
				videoPatterns := []string{
					filepath.Join(outputDir, "videos", fileName, "720p30", className+".mp4"),
					filepath.Join(outputDir, "videos", fileName, "1080p60", className+".mp4"),
					filepath.Join(outputDir, "videos", fileName, "480p30", className+".mp4"),
					filepath.Join(outputDir, "videos", "animation", "720p30", className+".mp4"),
					filepath.Join(outputDir, "videos", "animation", "1080p60", className+".mp4"),
					filepath.Join(outputDir, "videos", "animation", "480p30", className+".mp4"),
					// 添加更多可能的路径模式
					filepath.Join(outputDir, "videos", "720p30", className+".mp4"),
					filepath.Join(outputDir, "videos", "1080p60", className+".mp4"),
					filepath.Join(outputDir, "videos", "480p30", className+".mp4"),
				}

				for _, pattern := range videoPatterns {
					if _, err := os.Stat(pattern); err == nil {
						return pattern
					}
				}
			}
		}
	}

	return ""
}

// findVideoByPatterns 通过模式匹配查找视频文件
func (s *ManimService) findVideoByPatterns(outputDir, fileName string) string {
	possiblePatterns := []string{
		filepath.Join(outputDir, "videos", fileName, "720p30", "*.mp4"),
		filepath.Join(outputDir, "videos", fileName, "1080p60", "*.mp4"),
		filepath.Join(outputDir, "videos", fileName, "480p30", "*.mp4"),
		filepath.Join(outputDir, "videos", "animation", "720p30", "*.mp4"),
		filepath.Join(outputDir, "videos", "animation", "1080p60", "*.mp4"),
		filepath.Join(outputDir, "videos", "animation", "480p30", "*.mp4"),
		filepath.Join(outputDir, "videos", "720p30", "*.mp4"),
		filepath.Join(outputDir, "videos", "1080p60", "*.mp4"),
		filepath.Join(outputDir, "videos", "480p30", "*.mp4"),
		
		// 添加缺失的路径模式：包含"-"目录的路径（默认场景）
		filepath.Join(outputDir, "videos", "-", "720p30", "*.mp4"),
		filepath.Join(outputDir, "videos", "-", "1080p60", "*.mp4"),
		filepath.Join(outputDir, "videos", "-", "480p30", "*.mp4"),
		
		filepath.Join(outputDir, "videos", "*.mp4"),
		filepath.Join(outputDir, "*.mp4"),
	}

	for _, pattern := range possiblePatterns {
		files, _ := filepath.Glob(pattern)
		if len(files) > 0 {
			// 优先选择主视频文件（不是partial_movie_files）
			for _, file := range files {
				if !strings.Contains(file, "partial_movie_files") {
					return file
				}
			}
			// 如果没有找到主视频文件，使用第一个文件
			return files[0]
		}
	}

	return ""
}

// searchVideoRecursively 递归搜索视频文件
func (s *ManimService) searchVideoRecursively(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if result := s.searchVideoRecursively(fullPath); result != "" {
				return result
			}
		} else if strings.HasSuffix(entry.Name(), ".mp4") && !strings.Contains(entry.Name(), "partial_movie_files") {
			return fullPath
		}
	}

	return ""
}

// moveVideoToFinalLocation 移动视频到最终位置（不重命名）
func (s *ManimService) moveVideoToFinalLocation(videoPath, finalVideoPath string, videoID uint, manimCode string) error {
	// 首先检查源视频文件是否存在
	if _, err := os.Stat(videoPath); err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("源视频文件不存在: %v", err))
		return fmt.Errorf("源视频文件不存在: %v", err)
	}

	// 创建最终目录
	err := os.MkdirAll(filepath.Dir(finalVideoPath), 0755)
	if err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("创建最终目录失败: %v", err))
		return err
	}

	// 如果目标文件已存在，先删除它（确保覆盖）
	if _, err := os.Stat(finalVideoPath); err == nil {
		if err := os.Remove(finalVideoPath); err != nil {
			s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("删除旧视频文件失败: %v", err))
			return fmt.Errorf("删除旧视频文件失败: %v", err)
		}
	}

	// 移动文件（重命名）而不是复制，提高效率
	err = os.Rename(videoPath, finalVideoPath)
	if err != nil {
		// 如果移动失败，尝试复制
		err = copyFile(videoPath, finalVideoPath)
		if err != nil {
			s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("移动视频文件失败: %v", err))
			return fmt.Errorf("移动视频文件失败: %v", err)
		}
		// 复制成功后删除源文件
		os.Remove(videoPath)
	}

	// 验证移动后的文件是否存在
	if _, err := os.Stat(finalVideoPath); err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("移动后文件验证失败: %v", err))
		return fmt.Errorf("移动后文件验证失败: %v", err)
	}

	return nil
}

// copyVideoToFinalLocation 复制视频到最终位置
func (s *ManimService) copyVideoToFinalLocation(videoPath, finalVideoPath string, videoID uint, manimCode string) error {
	// 首先检查源视频文件是否存在
	if _, err := os.Stat(videoPath); err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("源视频文件不存在: %v", err))
		return fmt.Errorf("源视频文件不存在: %v", err)
	}

	// 创建最终目录
	err := os.MkdirAll(filepath.Dir(finalVideoPath), 0755)
	if err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("创建最终目录失败: %v", err))
		return err
	}

	// 如果目标文件已存在，先删除它（确保覆盖）
	if _, err := os.Stat(finalVideoPath); err == nil {
		if err := os.Remove(finalVideoPath); err != nil {
			s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("删除旧视频文件失败: %v", err))
			return fmt.Errorf("删除旧视频文件失败: %v", err)
		}
	}

	// 复制文件
	err = copyFile(videoPath, finalVideoPath)
	if err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("复制视频文件失败: %v", err))
		return err
	}

	// 验证复制后的文件是否存在
	if _, err := os.Stat(finalVideoPath); err != nil {
		s.videoService.UpdateVideoStatus(context.Background(), videoID, model.VideoStatusFailed, manimCode, "", fmt.Sprintf("复制后文件验证失败: %v", err))
		return fmt.Errorf("复制后文件验证失败: %v", err)
	}

	return nil
}

// copyFile 复制文件 - 改进版本，支持大文件
func copyFile(src, dst string) error {
	// 检查源文件是否存在
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("源文件不存在: %v", err)
	}

	// 检查源文件大小，如果文件过大使用流式复制
	if srcInfo.Size() > 100*1024*1024 { // 大于100MB的文件使用流式复制
		return copyFileStreaming(src, dst)
	}

	// 对于小文件，使用内存复制
	// 读取源文件
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// 写入目标文件
	err = os.WriteFile(dst, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// copyFileStreaming 流式复制大文件
func copyFileStreaming(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 使用缓冲区进行流式复制
	buffer := make([]byte, 32*1024) // 32KB缓冲区
	for {
		n, err := srcFile.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := dstFile.Write(buffer[:n]); err != nil {
			return err
		}
	}

	return nil
}

// CleanOldVirs 清理过期视频文件 - 修复函数名和逻辑
func (s *ManimService) CleanOldVirs(ctx context.Context, days int) error {
	videosDir := "videos"

	// 检查目录是否存在
	if _, err := os.Stat(videosDir); os.IsNotExist(err) {
		return nil // 目录不存在，无需清理
	}

	entries, err := os.ReadDir(videosDir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // 跳过文件，只处理目录
		}

		userDir := filepath.Join(videosDir, entry.Name())

		// 检查用户目录下的视频文件
		videoFiles, err := os.ReadDir(userDir)
		if err != nil {
			continue // 跳过无法读取的目录
		}

		for _, videoFile := range videoFiles {
			if videoFile.IsDir() {
				continue // 跳过子目录
			}

			filePath := filepath.Join(userDir, videoFile.Name())

			// 检查文件扩展名
			if !strings.HasSuffix(strings.ToLower(videoFile.Name()), ".mp4") {
				continue // 跳过非视频文件
			}

			fileInfo, err := videoFile.Info()
			if err != nil {
				continue
			}

			if fileInfo.ModTime().Before(cutoffTime) {
				if err := os.Remove(filePath); err != nil {
					// 记录错误但继续清理其他文件
					fmt.Printf("删除过期视频文件失败: %v\n", err)
				}
			}
		}
	}

	return nil
}

// ProcessPendingVideos 处理待处理的视频
func (s *ManimService) ProcessPendingVideos(ctx context.Context) error {
	videos, err := s.videoService.GetProcessingVideos(ctx)
	if err != nil {
		return err
	}

	for _, video := range videos {
		if video.Status == model.VideoStatusPending && video.ManimCode != "" {
			go func(v model.Video) {
				s.GenerateVideo(ctx, v.ID, v.ManimCode)
			}(video)
		}
	}

	return nil
}

// GetQueueStatus 获取队列状态
func (s *ManimService) GetQueueStatus() (int, int) {
	return len(s.semaphore), cap(s.semaphore)
}
