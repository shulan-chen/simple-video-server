package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
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

func homeHandler(c *gin.Context) {
	cname, err1 := c.Cookie("username")
	sid, err2 := c.Cookie("sessionid")
	if err1 == nil && err2 == nil {
		if cname != "" && sid != "" {
			c.Redirect(http.StatusFound, "/userhome")
			return
		}
	}
	c.HTML(http.StatusOK, "home.html", HomePage{Name: "unknown"})
}

func userHomeHandler(c *gin.Context) {
	cname, err1 := c.Cookie("username")
	sid, err2 := c.Cookie("sessionid")
	if err1 != nil || err2 != nil { // missing cookie ，redirect to root path
		c.Redirect(http.StatusFound, "/")
		return
	}
	if cname == "" || sid == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	var p *UserHomePage
	formUsername := c.PostForm("username")
	if len(cname) != 0 {
		p = &UserHomePage{Name: cname}
	} else if len(formUsername) != 0 {
		p = &UserHomePage{Name: formUsername}
	}

	c.HTML(http.StatusOK, "userhome.html", p)
}

func apiHandler(c *gin.Context) {
	if c.Request.Method != http.MethodPost {
		sendErrorResponse(c.Writer, ErrorRequestNotRecognized)
		return
	}
	apiBody := &ApiBody{}
	if err := c.BindJSON(apiBody); err != nil {
		sendErrorResponse(c.Writer, ErrorRequestBodyParseFailed)
		return
	}

	apiRequestProcess(apiBody, c.Writer, c.Request)
	defer c.Request.Body.Close()
}

func proxyUploadHandler(c *gin.Context) {
	u, _ := url.Parse("http://localhost:9090/")
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(c.Writer, c.Request)
}

func proxyVideoViewHandler(c *gin.Context) {
	u, _ := url.Parse("http://localhost:9090/")
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(c.Writer, c.Request)
}
