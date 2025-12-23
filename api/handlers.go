package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"video-server/api/dbops"
	api "video-server/api/defs"

	"github.com/julienschmidt/httprouter"
)

func CreateUser(w http.ResponseWriter, req *http.Request, param httprouter.Params) {

	res, _ := ioutil.ReadAll(req.Body)
	loginUser := &api.User{}

	err := json.Unmarshal(res, loginUser)
	if err != nil {
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
	}

	err = dbops.AddUser(loginUser.Username, loginUser.Password)

}
