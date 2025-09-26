import React, { useState, useEffect } from 'react'
import { videoAPI } from '../services/api'

const Profile = () => {
  const [videos, setVideos] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    loadVideos()
  }, [])

  const loadVideos = async () => {
    try {
      setLoading(true)
      const response = await videoAPI.getVideos({ page: 1, page_size: 50 })
      setVideos(response.videos || [])
    } catch (error) {
      setError('获取视频列表失败')
      console.error('Error loading videos:', error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusText = (status) => {
    const statusMap = {
      pending: '等待中',
      processing: '处理中',
      completed: '已完成',
      failed: '失败'
    }
    return statusMap[status] || status
  }

  const getStatusClass = (status) => {
    const classMap = {
      pending: 'status-pending',
      processing: 'status-processing',
      completed: 'status-completed',
      failed: 'status-failed'
    }
    return classMap[status] || 'status-pending'
  }

  const handleDownload = async (videoId, filename) => {
    try {
      const response = await videoAPI.downloadVideo(videoId)
      
      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response]))
      const link = document.createElement('a')
      link.href = url
      link.download = filename || `video_${videoId}.mp4`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (error) {
      console.error('下载失败:', error)
      alert('下载失败，请稍后重试')
    }
  }

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString('zh-CN')
  }

  if (loading) {
    return (
      <div>
        <div className="page-header">
          <div className="container">
            <h1>个人中心</h1>
            <p>查看您的动画生成记录</p>
          </div>
        </div>
        <div className="container">
          <div className="loading">
            <div>加载中...</div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="page-header">
        <div className="container">
          <h1>个人中心</h1>
          <p>查看您的动画生成记录</p>
        </div>
      </div>
      
      <div className="container">
        {error && <div className="error">{error}</div>}
        
        <div className="card">
          <h2>我的动画记录</h2>
          <p>共 {videos.length} 个动画</p>
          
          {videos.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '2rem', color: '#666' }}>
              暂无动画记录，去生成您的第一个动画吧！
            </div>
          ) : (
            <div className="video-list">
              {videos.map((video) => (
                <div key={video.id} className="video-item">
                  <div style={{ marginBottom: '1rem' }}>
                    <h3 style={{ marginBottom: '0.5rem' }}>
                      {video.prompt || '未命名动画'}
                    </h3>
                    <div style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between', 
                      alignItems: 'center',
                      marginBottom: '0.5rem'
                    }}>
                      <span className={`status-badge ${getStatusClass(video.status)}`}>
                        {getStatusText(video.status)}
                      </span>
                      <span style={{ fontSize: '0.875rem', color: '#666' }}>
                        {formatDate(video.created_at)}
                      </span>
                    </div>
                    
                    {video.error_message && (
                      <div style={{ 
                        fontSize: '0.875rem', 
                        color: '#ff4d4f',
                        marginBottom: '0.5rem'
                      }}>
                        错误: {video.error_message}
                      </div>
                    )}
                  </div>
                  
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    {video.status === 'completed' && video.video_url && (
                      <button 
                        onClick={() => handleDownload(video.id, `animation_${video.id}.mp4`)}
                        className="btn-primary"
                        style={{ flex: 1 }}
                      >
                        下载视频
                      </button>
                    )}
                    
                    {video.status === 'completed' && video.video_url && (
                      <button 
                        onClick={() => window.open(video.video_url, '_blank')}
                        className="btn-secondary"
                        style={{ flex: 1 }}
                      >
                        在线预览
                      </button>
                    )}
                    
                    {(video.status === 'pending' || video.status === 'processing') && (
                      <button 
                        className="btn-secondary"
                        style={{ flex: 1 }}
                        disabled
                      >
                        处理中...
                      </button>
                    )}
                    
                    {video.status === 'failed' && (
                      <button 
                        className="btn-secondary"
                        style={{ flex: 1 }}
                        disabled
                      >
                        生成失败
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Profile