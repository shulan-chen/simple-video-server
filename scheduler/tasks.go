package scheduler

import (
	"context"
	"fmt"
	"strings"
	"video-server/api/dbops"
	"video-server/api/utils"
	"video-server/stream"

	"go.uber.org/zap"
)

// VideoClearDispatcher 从数据库读取待删除的视频 ID 并发送到 channel
// 职责：批量读取任务，推送到执行队列
// 返回：nil 表示成功（包括暂无任务的情况），error 表示真正的错误
func (r *Runner) VideoClearDispatcher(dc dataChannel) error {
	vids, err := dbops.ReadVideoDeletionRecord(r.dataSize)
	if err != nil {
		utils.Logger.Error("读取待删除视频记录失败", zap.Error(err))
		return err
	}
	if len(vids) == 0 {
		// 暂无任务是正常情况，返回特殊error，其实是个信号
		return fmt.Errorf("暂无待删除视频")
	}
	for _, vid := range vids {
		dc <- vid
	}
	utils.Logger.Debug("成功分发待删除视频任务", zap.Int("count", len(vids)))
	return nil
}

// VideoClearExecutor 执行视频删除任务（事务性删除）
// 职责：执行删除逻辑，保证事务性和可回滚
func (r *Runner) VideoClearExecutor(dc dataChannel) error {
	var (
		successCount int
		failedVids   []string
		lastErr      error
	)

	ctx := context.Background()

drainLoop:
	for {
		select {
		case vid := <-dc:
			vidStr := vid.(string)

			// 执行事务性删除
			if err := deleteVideoTransaction(ctx, vidStr); err != nil {
				utils.Logger.Error("删除视频失败",
					zap.String("vid", vidStr),
					zap.Error(err))
				failedVids = append(failedVids, vidStr)
				lastErr = err
			} else {
				successCount++
				utils.Logger.Info("成功删除视频",
					zap.String("vid", vidStr))
			}

		default:
			break drainLoop
		}
	}
	// 汇总执行结果
	if successCount > 0 || len(failedVids) > 0 {
		utils.Logger.Info("视频删除任务执行完成",
			zap.Int("成功", successCount),
			zap.Int("失败", len(failedVids)))
	}
	if len(failedVids) > 0 {
		return fmt.Errorf("部分视频删除失败 [%s]: %w",
			strings.Join(failedVids, ", "), lastErr)
	}
	return nil
}

// deleteVideoTransaction 执行单个视频的删除事务（支持回滚）
//
// 事务流程：
//  1. 软删除数据库记录（设置 deleted_at）
//  2. 删除 OSS 视频文件
//  3. 删除 OSS 封面文件
//  4. 删除视频评论
//  5. 成功 → 物理删除数据库记录
//     失败 → 恢复软删除（清空 deleted_at）
func deleteVideoTransaction(ctx context.Context, vid string) error {
	// 步骤1：软删除记录（GORM 标准逻辑删除）
	if err := dbops.SoftDeleteVideoDeletionRecord(vid); err != nil {
		return fmt.Errorf("软删除记录失败: %w", err)
	}

	// 步骤2：删除 OSS 视频文件
	if err := stream.DeleteFromOSS(ctx, vid); err != nil {
		// OSS 删除失败 → 恢复软删除（回滚）
		if rollbackErr := dbops.RestoreVideoDeletionRecord(vid); rollbackErr != nil {
			utils.Logger.Error("回滚软删除失败",
				zap.String("vid", vid),
				zap.Error(rollbackErr))
			return fmt.Errorf("OSS视频删除失败且回滚失败: oss_err=%w, rollback_err=%v", err, rollbackErr)
		}

		utils.Logger.Warn("OSS 视频删除失败，已回滚，等待下次重试",
			zap.String("vid", vid),
			zap.Error(err))
		return fmt.Errorf("OSS视频删除失败: %w", err)
	}

	// 步骤3：删除 OSS 封面文件（titlePage/{vid}.jpg）
	if err := stream.DeleteThumbnailFromOSS(ctx, vid); err != nil {
		// 封面删除失败，记录日志但不回滚（视频文件已删除）
		utils.Logger.Warn("OSS 封面删除失败（视频已删除）",
			zap.String("vid", vid),
			zap.Error(err))
		// 继续执行，不阻塞整个流程
	}

	// 步骤4：OSS 删除成功 → 物理删除数据库记录
	if err := dbops.DeleteVideoDeletionRecord(vid); err != nil {
		utils.Logger.Error("物理删除记录失败（OSS已删除）",
			zap.String("vid", vid),
			zap.Error(err))
		return fmt.Errorf("物理删除记录失败: %w", err)
	}

	return nil
}
