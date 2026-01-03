package stream

import (
	"io"
	"net/http"
)

func sendErrorResponse(w http.ResponseWriter, sc int, errMsg string) {
	w.WriteHeader(sc)
	io.WriteString(w, errMsg)
}
