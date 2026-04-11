-- 迁移脚本：为 video_delete_record 表添加 GORM 标准逻辑删除字段
-- 日期：2026-04-11
-- 目的：使用 GORM 标准逻辑删除，实现事务性删除

-- 步骤1：添加时间字段
ALTER TABLE video_delete_record
ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL COMMENT 'GORM 标准逻辑删除字段';

-- 步骤2：为 vid 字段添加索引（如果还没有）
ALTER TABLE video_delete_record
ADD INDEX IF NOT EXISTS idx_vid (vid);

-- 步骤3：为 deleted_at 字段添加索引（GORM 标准）
ALTER TABLE video_delete_record
ADD INDEX idx_deleted_at (deleted_at);

-- 查看表结构
DESC video_delete_record;

-- 验证数据
SELECT
    COUNT(*) as total,
    COUNT(deleted_at) as soft_deleted,
    COUNT(CASE WHEN deleted_at IS NULL THEN 1 END) as active
FROM video_delete_record;
