-- 回滚：删除 deleted_at 字段

-- 评论表
DROP INDEX idx_comments_deleted ON comments;
ALTER TABLE comments DROP COLUMN deleted_at;

-- 视频表
DROP INDEX idx_video_deleted ON video_info;
ALTER TABLE video_info DROP COLUMN deleted_at;

-- 用户表
DROP INDEX idx_users_deleted ON users;
ALTER TABLE users DROP COLUMN deleted_at;
