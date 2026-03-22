-- 先删除旧表（如果存在）
DROP TABLE IF EXISTS video_delete_record;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS video_info;
DROP TABLE IF EXISTS users;

-- 用户表
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(32) UNIQUE NOT NULL,
    password VARCHAR(128) NOT NULL,
    is_valid TINYINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_users_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 视频表
CREATE TABLE video_info (
    id INT AUTO_INCREMENT PRIMARY KEY,
    vid VARCHAR(64) UNIQUE NOT NULL,
    author_id INT NOT NULL,
    name VARCHAR(128) NOT NULL,
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    click_count INT DEFAULT 0,
    INDEX idx_video_vid (vid),
    INDEX idx_video_author (author_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 评论表
CREATE TABLE comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    comment_id VARCHAR(64) UNIQUE NOT NULL,
    video_id VARCHAR(64) NOT NULL,
    author_id INT NOT NULL,
    content TEXT NOT NULL,
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_comment_video (video_id),
    INDEX idx_comment_id (comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Session表
CREATE TABLE sessions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(128) UNIQUE NOT NULL,
    user_id INT NOT NULL,
    username VARCHAR(32) NOT NULL,
    ttl VARCHAR(32),
    INDEX idx_session_id (session_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 视频删除记录表
CREATE TABLE video_delete_record (
    id INT AUTO_INCREMENT PRIMARY KEY,
    vid VARCHAR(64) NOT NULL,
    INDEX idx_vdr_vid (vid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;