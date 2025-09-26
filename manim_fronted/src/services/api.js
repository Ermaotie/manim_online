import axios from 'axios'

const API_BASE_URL = '/api'

// 创建axios实例
const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 60000, // 增加超时时间到60秒，适应视频渲染需求
})

// 请求拦截器 - 添加token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器 - 处理错误
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    }
    return Promise.reject(error.response?.data || error)
  }
)

// 认证相关API
export const authAPI = {
  // 用户注册
  register: (userData) => api.post('/auth/register', userData),
  
  // 用户登录
  login: (credentials) => api.post('/auth/login', credentials),
  
  // 获取用户信息
  getProfile: () => api.get('/user/profile'),
}

// AI相关API
export const aiAPI = {
  // 生成Manim代码
  generateCode: (prompt) => api.post('/ai/generate', { prompt }),
  
  // 验证Manim代码
  validateCode: (code) => api.post('/ai/validate', { code }),
}

// 视频相关API
export const videoAPI = {
  // 创建视频任务
  createVideo: (videoData) => api.post('/videos', videoData),
  
  // 获取视频列表
  getVideos: (params = {}) => api.get('/videos', { params }),
  
  // 获取视频详情
  getVideoDetail: (id) => api.get('/videos/detail', { params: { id } }),
  
  // 下载视频
  downloadVideo: (id) => api.get(`/videos/${id}/download`, { responseType: 'blob' }),
}

// 健康检查
export const healthAPI = {
  check: () => api.get('/health'),
}

export default api