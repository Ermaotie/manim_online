import React, { useState } from 'react'
import { aiAPI, videoAPI } from '../services/api'

const Home = () => {
  const [prompt, setPrompt] = useState('')
  const [generatedCode, setGeneratedCode] = useState('')
  const [videoStatus, setVideoStatus] = useState('idle') // idle, generating, completed, error
  const [currentVideo, setCurrentVideo] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const handleGenerateCode = async () => {
    if (!prompt.trim()) {
      setError('请输入描述文字')
      return
    }

    setLoading(true)
    setError('')
    setSuccess('')
    setGeneratedCode('')
    setVideoStatus('generating')

    try {
      // 直接调用 /api/videos 接口创建视频任务
      const videoData = {
        prompt: prompt,
        title: `动画 - ${prompt.substring(0, 30)}...`
      }
      
      const response = await videoAPI.createVideo(videoData)
      setGeneratedCode(response.manimCode)
      setCurrentVideo(response)
      setVideoStatus('processing')
      setSuccess('视频任务已创建，正在处理中...')
      
      // 开始轮询视频状态
      pollVideoStatus(response.id)
    } catch (error) {
      setError('代码生成失败：' + (error.message || '未知错误'))
      setVideoStatus('error')
      setLoading(false)
    }
  }

  const handleCreateVideo = async () => {
    if (!generatedCode) {
      setError('请先生成代码')
      return
    }

    setLoading(true)
    setError('')
    setSuccess('')
    setVideoStatus('generating')

    try {
      // 先验证代码
      await aiAPI.validateCode(generatedCode)
      
      // 创建视频任务
      const videoData = {
        prompt: prompt,
        code: generatedCode,
        title: `动画 - ${prompt.substring(0, 30)}...`
      }
      
      const response = await videoAPI.createVideo(videoData)
      setCurrentVideo(response)
      setVideoStatus('processing')
      setSuccess('视频任务已创建，正在处理中...')
      
      // 开始轮询视频状态
      pollVideoStatus(response.id)
    } catch (error) {
      setError('视频创建失败：' + (error.message || '未知错误'))
      setVideoStatus('error')
      setLoading(false)
    }
  }

  const pollVideoStatus = async (videoId) => {
    const maxAttempts = 120 // 增加轮询次数到120次（最长4分钟）
    let attempts = 0
    let lastErrorTime = 0
    const errorRetryDelay = 5000 // 错误重试延迟5秒

    const checkStatus = async () => {
      try {
        const video = await videoAPI.getVideoDetail(videoId)
        
        if (video.status === 'completed') {
          setVideoStatus('completed')
          setCurrentVideo(video)
          setSuccess('视频生成完成！')
          setLoading(false)
          return
        } else if (video.status === 'failed') {
          setVideoStatus('error')
          setError('视频生成失败：' + (video.error_message || '未知错误'))
          setLoading(false)
          return
        }
        
        // 继续轮询
        attempts++
        if (attempts < maxAttempts) {
          // 动态调整轮询间隔：开始时2秒，逐渐增加到5秒
          const interval = Math.min(2000 + Math.floor(attempts / 10) * 1000, 5000)
          setTimeout(checkStatus, interval)
        } else {
          setVideoStatus('error')
          setError('视频生成超时（最长等待4分钟），请稍后查看个人中心')
          setLoading(false)
        }
      } catch (error) {
        console.error('轮询视频状态失败:', error)
        
        // 错误处理：如果是网络错误，增加重试延迟
        const now = Date.now()
        if (now - lastErrorTime < 30000) { // 30秒内连续错误
          attempts += 3 // 快速消耗尝试次数
        }
        lastErrorTime = now
        
        attempts++
        if (attempts < maxAttempts) {
          // 错误时使用更长的重试间隔
          setTimeout(checkStatus, errorRetryDelay)
        } else {
          setVideoStatus('error')
          setError('获取视频状态失败，请检查网络连接')
          setLoading(false)
        }
      }
    }

    checkStatus()
  }

  const handleDownloadVideo = async () => {
    if (!currentVideo) return
    
    try {
      const response = await videoAPI.downloadVideo(currentVideo.id)
      
      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response]))
      const link = document.createElement('a')
      link.href = url
      link.download = `animation_${currentVideo.id}.mp4`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (error) {
      console.error('下载失败:', error)
      alert('下载失败，请稍后重试')
    }
  }

  const handleReset = () => {
    setPrompt('')
    setGeneratedCode('')
    setVideoStatus('idle')
    setCurrentVideo(null)
    setError('')
    setSuccess('')
  }

  return (
    <div>
      <div className="page-header">
        <div className="container">
          <h1>Manim动画生成平台</h1>
          <p>使用AI技术，将您的想法转化为精美的数学动画</p>
        </div>
      </div>
      
      <div className="container">
        {error && <div className="error">{error}</div>}
        {success && <div className="success">{success}</div>}
        
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '2rem', alignItems: 'start' }}>
          {/* 左侧：输入区域 */}
          <div className="card">
            <h2>描述您的动画想法</h2>
            <div className="form-group">
              <textarea
                className="form-control textarea"
                placeholder="例如：创建一个展示勾股定理的动画，包含直角三角形和三个正方形..."
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                disabled={loading}
                rows={6}
              />
            </div>
            
            <div style={{ display: 'flex', gap: '1rem' }}>
              <button 
                onClick={handleGenerateCode}
                className="btn-primary"
                disabled={loading || !prompt.trim()}
              >
                {loading && !generatedCode ? '生成中...' : '生成Manim代码'}
              </button>
            </div>
          </div>
          
          {/* 右侧：动画展示区域 */}
          <div className="card">
            <h2>动画预览</h2>
            <div style={{ 
              minHeight: '300px', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              background: '#f8f9fa',
              borderRadius: '8px',
              padding: '2rem'
            }}>
              {videoStatus === 'idle' && !generatedCode && (
                <div style={{ textAlign: 'center', color: '#666' }}>
                  <div style={{ fontSize: '4rem', marginBottom: '1rem' }}>🎬</div>
                  <p>请输入动画描述并生成代码</p>
                  <p style={{ fontSize: '0.875rem', marginTop: '0.5rem' }}>生成的动画将在这里展示</p>
                </div>
              )}
              
              {generatedCode && videoStatus === 'processing' && (
                <div style={{ textAlign: 'center', color: '#666' }}>
                  <div style={{ fontSize: '4rem', marginBottom: '1rem' }}>📝</div>
                  <p>代码已生成，视频正在处理中...</p>
                </div>
              )}
              
              {videoStatus === 'generating' && (
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>⏳</div>
                  <p>正在创建视频任务...</p>
                </div>
              )}
              
              {videoStatus === 'processing' && (
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>🔧</div>
                  <p>视频正在处理中，请稍候...</p>
                  <p style={{ fontSize: '0.875rem', color: '#666' }}>
                    这可能需要几分钟时间
                  </p>
                </div>
              )}
              
              {videoStatus === 'completed' && currentVideo && currentVideo.video_path && (
                <div style={{ width: '100%', maxWidth: '500px' }}>
                  <video 
                    controls 
                    autoPlay
                    muted
                    style={{ 
                      width: '100%', 
                      borderRadius: '8px',
                      boxShadow: '0 4px 12px rgba(0,0,0,0.1)'
                    }}
                    src={`http://localhost:8888/downloadvideo${currentVideo.video_path.replace(/\\/g, '/').replace('/videos/', '/')}`}
                  >
                    您的浏览器不支持视频播放
                  </video>
                  
                  <div style={{ 
                    display: 'flex', 
                    gap: '1rem', 
                    justifyContent: 'center',
                    marginTop: '1rem'
                  }}>
                    <button 
                      onClick={handleDownloadVideo}
                      className="btn-primary"
                    >
                      下载视频
                    </button>
                    
                    <button 
                      onClick={() => window.open(`http://localhost:8888/downloadvideo${currentVideo.video_path.replace(/\\/g, '/').replace('/videos/', '/')}`, '_blank')}
                      className="btn-secondary"
                    >
                      在新窗口打开
                    </button>
                  </div>
                </div>
              )}
              
              {videoStatus === 'error' && (
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>❌</div>
                  <p>视频生成失败</p>
                  <p style={{ fontSize: '0.875rem', color: '#666', marginTop: '0.5rem' }}>
                    {currentVideo?.error_msg || '请检查您的输入或稍后重试'}
                  </p>
                  <button 
                    onClick={handleReset}
                    className="btn-primary"
                    style={{ marginTop: '1rem' }}
                  >
                    重新尝试
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        {generatedCode && (
          <div className="card" style={{ gridColumn: '1 / -1' }}>
            <h2>生成的Manim代码</h2>
            <div style={{ 
              background: '#f8f9fa', 
              padding: '1rem', 
              borderRadius: '6px',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
              whiteSpace: 'pre-wrap',
              maxHeight: '300px',
              overflow: 'auto',
              border: '1px solid #e9ecef'
            }}>
              {generatedCode}
            </div>
            
            <div style={{ marginTop: '1rem', display: 'flex', gap: '1rem' }}>
              <button 
                onClick={handleCreateVideo}
                className="btn-primary"
                disabled={loading || videoStatus === 'generating' || videoStatus === 'processing'}
              >
                {videoStatus === 'generating' ? '创建任务中...' : 
                 videoStatus === 'processing' ? '处理中...' : 
                 '生成视频'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default Home