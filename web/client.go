package web

import (
	"io"
	"net/http"
)

var httpClient *http.Client

func init() {
	httpClient = &http.Client{}
}

func apiRequestProcess(apiBody *ApiBody, w http.ResponseWriter, req *http.Request) {
	var resp *http.Response
	//var err error

	switch apiBody.Method {
	case "GET":
		netRequest, err := http.NewRequest("GET", apiBody.Url, nil)
		netRequest.Header = req.Header
		resp, err = httpClient.Do(netRequest)
		if err != nil {
			sendErrorResponse(w, ErrorInternalFaults)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, val := range v {
				w.Header().Add(k, val)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	case http.MethodPost:
		netRequest, err := http.NewRequest("POST", apiBody.Url, req.Body)
		netRequest.Header = req.Header
		resp, err = httpClient.Do(netRequest)
		if err != nil {
			sendErrorResponse(w, ErrorInternalFaults)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, val := range v {
				w.Header().Add(k, val)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	case http.MethodDelete:
		netRequest, err := http.NewRequest("DELETE", apiBody.Url, nil)
		netRequest.Header = req.Header
		resp, err = httpClient.Do(netRequest)
		if err != nil {
			sendErrorResponse(w, ErrorInternalFaults)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, val := range v {
				w.Header().Add(k, val)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	default:
		sendErrorResponse(w, ErrorRequestNotRecognized)
		return
	}
}
