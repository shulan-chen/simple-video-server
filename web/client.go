package web

import (
	"io"
	"net/http"
	"strings"
)

var httpClient *http.Client

func init() {
	httpClient = &http.Client{}
}

func apiRequestProcess(apiBody *ApiBody, w http.ResponseWriter, req *http.Request) {
	var resp *http.Response
	//var err error

	switch apiBody.Method {
	case http.MethodGet:
		netRequest, err := http.NewRequest("GET", apiBody.Url, nil)
		netRequest.Header = req.Header
		resp, err = httpClient.Do(netRequest)
		if err != nil {
			sendErrorResponse(w, ErrorInternalProxyFaults)
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
		netRequest, err := http.NewRequest("POST", apiBody.Url, strings.NewReader(apiBody.ReqBody))
		netRequest.Header = req.Header
		netRequest.Header.Del("Content-Length")
		resp, err = httpClient.Do(netRequest)
		if err != nil {
			sendErrorResponse(w, ErrorInternalProxyFaults)
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
			sendErrorResponse(w, ErrorInternalProxyFaults)
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
