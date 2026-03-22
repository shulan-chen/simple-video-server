-- 为需要软删除的表添加 deleted_at 字段

-- 用户表
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;
CREATE INDEX idx_users_deleted ON users(deleted_at);

-- 视频表
ALTER TABLE video_info ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;
CREATE INDEX idx_video_deleted ON video_info(deleted_at);

-- 评论表
ALTER TABLE comments ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;
CREATE INDEX idx_comments_deleted ON comments(deleted_at);
