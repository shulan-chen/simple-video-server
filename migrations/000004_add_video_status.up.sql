ALTER TABLE video_info
  ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'available'
  COMMENT 'pending/processing/available/rejected/failed';

CREATE INDEX idx_video_status ON video_info(status);
