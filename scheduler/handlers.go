package scheduler

import (
	"net/http"
	"video-server/api/dbops"

	"github.com/julienschmidt/httprouter"
)

func vidDelRecHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	vid := ps.ByName("vid-vid")
	if len(vid) == 0 {
		http.Error(w, "Video id should not be empty", http.StatusBadRequest)
		return
	}
	err := dbops.InsertNewVideoDeletionRecord(vid)
	if err != nil {
		http.Error(w, "Failed to schedule video delete record task", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Video delete record task scheduled successfully"))
}
