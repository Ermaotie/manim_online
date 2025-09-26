-- Manim 在线平台数据库初始化脚本
-- 数据库名: manim_db
-- 创建时间: $(date)

-- 创建数据库
CREATE DATABASE IF NOT EXISTS manim_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE manim_db;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_username (username),
    INDEX idx_email (email),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建视频表
CREATE TABLE IF NOT EXISTS videos (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    prompt TEXT NOT NULL,
    manim_code LONGTEXT,
    video_path VARCHAR(500),
    status TINYINT DEFAULT 0 COMMENT '0: pending, 1: processing, 2: completed, 3: failed',
    error_msg TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_deleted_at (deleted_at),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入测试用户数据（可选）
INSERT IGNORE INTO users (username, email, password) VALUES 
('testuser', 'test@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'), -- 密码: password
('admin', 'admin@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'); -- 密码: password

-- 插入测试视频数据（可选）
INSERT IGNORE INTO videos (user_id, title, description, prompt, manim_code, status) VALUES 
(1, '测试视频1', '这是一个测试视频', '创建一个简单的动画', 'from manim import *\n\nclass TestScene(Scene):\n    def construct(self):\n        circle = Circle()\n        self.play(Create(circle))', 2),
(1, '测试视频2', '另一个测试视频', '创建旋转的正方形', 'from manim import *\n\nclass SquareScene(Scene):\n    def construct(self):\n        square = Square()\n        self.play(Rotate(square, angle=PI))', 0);

-- 创建数据库用户（可选，根据实际需要配置）
-- CREATE USER IF NOT EXISTS 'manim_user'@'%' IDENTIFIED BY 'manim_password';
-- GRANT ALL PRIVILEGES ON manim_db.* TO 'manim_user'@'%';
-- FLUSH PRIVILEGES;

-- 显示表结构
SHOW TABLES;
DESCRIBE users;
DESCRIBE videos;