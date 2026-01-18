package taskrunner

import (
	"errors"
	"os"
	"sync"
	"video-server/api/dbops"
	"video-server/api/utils"
	"video-server/stream"

	"go.uber.org/zap"
)

var readNumber int = 3

func VideoClearDispatcher(dc dataChannel) error {
	// read from db
	vids, err := dbops.ReadVideoDeletionRecord(readNumber)
	if err != nil {
		//utils.Logger.Error("VideoClearDispatcher ReadVideoDeletionRecord failed", zap.Error(err))
		return err
	}
	if len(vids) == 0 {
		return errors.New("All tasks finished")
	}
	for _, vid := range vids {
		dc <- vid
	}
	return nil
}

func VideoClearExecutor(dc dataChannel) error {
	errMap := &sync.Map{}
	var err error

forloop:
	for {
		select {
		case vid := <-dc:
			// delete video file
			err = deleteVideoFile(vid.(string))
			if err != nil {
				utils.Logger.Error("VideoClearExecutor DeleteVideoFile failed", zap.String("vid", vid.(string)), zap.Error(err))
				errMap.Store(vid, err)
				continue
			}
			// delete db record
			err = dbops.DeleteVideoDeletionRecord(vid.(string))
			if err != nil {
				utils.Logger.Error("VideoClearExecutor DeleteVideoDeletionRecord failed", zap.String("vid", vid.(string)), zap.Error(err))
				errMap.Store(vid, err)
				continue
			}
			utils.Logger.Info("VideoClearExecutor successfully deleted video", zap.String("vid", vid.(string)))
		default:
			break forloop
		}
	}
	errMap.Range(func(k, v interface{}) bool {
		err = v.(error)
		if err != nil {
			return false
		}
		return true
	})

	return err
}

func deleteVideoFile(vid string) error {

	err := os.Remove(stream.VIDEO_DIR + vid)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
