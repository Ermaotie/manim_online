import React, { useState, useRef } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

const Login = () => {
  const [formData, setFormData] = useState({
    username: '',
    password: ''
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const errorRef = useRef('') // 使用ref来持久化错误信息
  
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  
  const from = location.state?.from?.pathname || '/'

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value
    })
    // 保持错误信息显示，不自动清除
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('') // 开始新的登录尝试时清除错误
    errorRef.current = '' // 同时清除ref中的错误信息

    try {
      const result = await login(formData)
      
      if (result.success) {
        navigate(from, { replace: true })
      } else {
        console.log('登录失败结果:', result)
        const errorMessage = result.error || '登录失败，请检查用户名和密码'
        setError(errorMessage)
        errorRef.current = errorMessage
      }
    } catch (error) {
      console.error('登录错误:', error)
      console.error('错误详情:', error.response || error)
      let errorMessage = '登录失败，请检查网络连接和服务器状态'
      
      if (error.code === 'ECONNABORTED') {
        errorMessage = '请求超时，请检查网络连接或稍后重试'
      } else if (error.response?.status === 404) {
        errorMessage = '无法连接到服务器，请确保后端服务正在运行'
      } else if (error.response?.status >= 500) {
        errorMessage = '服务器内部错误，请稍后重试'
      }
      
      setError(errorMessage)
      errorRef.current = errorMessage
    } finally {
      setLoading(false)
    }
  }

  // 添加重置错误的方法
  const resetError = () => {
    setError('')
    errorRef.current = ''
  }

  // 使用useEffect确保错误信息在重新渲染时保持同步
  React.useEffect(() => {
    // 如果ref中有错误信息但state中没有，则同步state
    if (errorRef.current && !error) {
      setError(errorRef.current)
    }
  }, [error])

  return (
    <div>
      <div className="page-header">
        <div className="container">
          <h1>登录</h1>
          <p>欢迎回到Manim动画平台</p>
        </div>
      </div>
      
      <div className="container">
        <div className="card" style={{ maxWidth: '400px', margin: '0 auto' }}>
          <form onSubmit={handleSubmit}>
            {error && (
              <div className="error" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>{error}</span>
                <button 
                  type="button" 
                  onClick={() => setError('')}
                  style={{ 
                    background: 'none', 
                    border: 'none', 
                    color: '#a8071a', 
                    cursor: 'pointer',
                    fontSize: '1.2rem'
                  }}
                >
                  ×
                </button>
              </div>
            )}
            
            <div className="form-group">
              <label htmlFor="username">用户名</label>
              <input
                type="text"
                id="username"
                name="username"
                className="form-control"
                value={formData.username}
                onChange={handleChange}
                required
              />
            </div>
            
            <div className="form-group">
              <label htmlFor="password">密码</label>
              <input
                type="password"
                id="password"
                name="password"
                className="form-control"
                value={formData.password}
                onChange={handleChange}
                required
              />
            </div>
            
            <button 
              type="submit" 
              className="btn-primary" 
              style={{ width: '100%' }}
              disabled={loading}
            >
              {loading ? '登录中...' : '登录'}
            </button>
            
            <div style={{ textAlign: 'center', marginTop: '1rem' }}>
              <span>还没有账号？</span>
              <Link to="/register" style={{ marginLeft: '0.5rem' }}>
                立即注册
              </Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default Login