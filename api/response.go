package api

import (
	"encoding/json"
	"io"
	"net/http"
	api "video-server/api/defs"
)

func sendErrorResponse(w http.ResponseWriter, errResp api.ErrResponse) {
	w.WriteHeader(errResp.HttpSC)

	resStr, _ := json.Marshal(&errResp.Error)
	io.WriteString(w, string(resStr))
}

func sendNormalResponse(w http.ResponseWriter, resp string, sc int) {
	w.WriteHeader(sc)
	io.WriteString(w, resp)
}
