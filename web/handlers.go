package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"text/template"
	"video-server/api/utils"

	"github.com/julienschmidt/httprouter"
)

type HomePage struct {
	Name string
}

type UserHomePage struct {
	Name string
}

func sendErrorResponse(w http.ResponseWriter, errResp ErrResponse) {
	w.WriteHeader(errResp.HttpSC)

	resStr, _ := json.Marshal(&errResp.Error)
	io.WriteString(w, string(resStr))
}

func sendNormalResponse(w http.ResponseWriter, resp string, sc int) {
	w.WriteHeader(sc)
	io.WriteString(w, resp)
}

func homeHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	cname, err1 := req.Cookie("username")
	sid, err2 := req.Cookie("sessionid")
	if err1 == nil && err2 == nil {
		if cname.Value != "" && sid.Value != "" {
			http.Redirect(w, req, "/userhome", http.StatusFound)
			return
		}
	}

	t, err := template.ParseFiles("./templates/home.html")
	if err != nil {
		utils.Logger.Error("Parsing template home.html faild")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, HomePage{Name: "unknown"})
}

func userHomeHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	cname, err1 := req.Cookie("username")
	sid, err2 := req.Cookie("sessionid")
	if err1 != nil || err2 != nil { // missing cookie ，redirect t
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}

	if cname.Value == "" || sid.Value == "" {
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}
	var p *UserHomePage
	formUsername := req.FormValue("username")
	if len(cname.Value) != 0 {
		p = &UserHomePage{Name: cname.Value}
	} else if len(formUsername) != 0 {
		p = &UserHomePage{Name: formUsername}
	}

	t, err := template.ParseFiles("./templates/userhome.html")
	if err != nil {
		utils.Logger.Error("Parsing template userhome.html faild")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, p)
}

func apiHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	if req.Method != http.MethodPost {
		sendErrorResponse(w, ErrorRequestNotRecognized)
		return
	}
	apiBody := &ApiBody{}
	reqBody, _ := io.ReadAll(req.Body)
	err := json.Unmarshal(reqBody, apiBody)
	if err != nil {
		utils.Logger.Error("Parsing api request body failed")
		sendErrorResponse(w, ErrorRequestBodyParseFailed)
		return
	}

	apiRequestProcess(apiBody, w, req)
	defer req.Body.Close()
}

func proxyHandler(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	u, _ := url.Parse("http://localhost:9090/")
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(w, req)
}
