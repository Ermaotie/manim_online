import React from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

const Navbar = () => {
  const { user, logout } = useAuth()
  const location = useLocation()

  const handleLogout = () => {
    logout()
  }

  const getInitials = (name) => {
    return name
      .split(' ')
      .map(word => word.charAt(0))
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  return (
    <nav className="navbar">
      <Link to="/" className="navbar-brand">
        Manim动画平台
      </Link>
      
      <div className="navbar-nav">
        <Link 
          to="/" 
          className={`nav-link ${location.pathname === '/' ? 'active' : ''}`}
        >
          生成动画
        </Link>
        <Link 
          to="/profile" 
          className={`nav-link ${location.pathname === '/profile' ? 'active' : ''}`}
        >
          个人中心
        </Link>
        
        {user ? (
          <div className="user-info">
            <div className="avatar">
              {getInitials(user.username || user.email)}
            </div>
            <span>{user.username || user.email}</span>
            <button 
              onClick={handleLogout}
              className="btn-secondary"
              style={{ marginLeft: '1rem' }}
            >
              退出登录
            </button>
          </div>
        ) : (
          <div className="user-info">
            <Link to="/login" className="btn-secondary">
              登录
            </Link>
            <Link to="/register" className="btn-primary">
              注册
            </Link>
          </div>
        )}
      </div>
    </nav>
  )
}

export default Navbar