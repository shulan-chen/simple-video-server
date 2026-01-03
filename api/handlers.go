package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"video-server/api/dbops"
	api "video-server/api/defs"
	"video-server/api/session"
	"video-server/api/utils"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

func CreateUser(w http.ResponseWriter, req *http.Request, param httprouter.Params) {

	reqBody, _ := ioutil.ReadAll(req.Body)
	loginUser := &api.User{}

	err := json.Unmarshal(reqBody, loginUser)
	if err != nil {
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
		return
	}

	_, err = dbops.AddUser(loginUser.Username, loginUser.Password)
	if err != nil {
		utils.Logger.Error("AddUser failed", zap.Error(err))
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	sendNormalResponse(w, "register success", 201)
}

func Login(w http.ResponseWriter, req *http.Request, param httprouter.Params) {
	uname := param.ByName("user_name")
	res, _ := ioutil.ReadAll(req.Body)
	loginUser := &api.User{}

	err := json.Unmarshal(res, loginUser)
	if err != nil {
		sendErrorResponse(w, api.ErrorRequestBodyParseFailed)
		return
	}
	loginUser.Username = uname

	pUser, err := dbops.GetUserByName(uname)
	if err != nil {
		sendErrorResponse(w, api.ErrorDBError)
		return
	}
	if pUser.Password != loginUser.Password {
		sendErrorResponse(w, api.ErrorNotAuthUser)
		return
	}

	simplesession := session.AddNewSession(pUser.Id, pUser.Username)
	su := api.SignedUP{
		SessionId: simplesession.SessionId,
		Success:   true,
	}
	if resp, err := json.Marshal(su); err != nil {
		sendErrorResponse(w, api.ErrorInternalFaults)
		return
	} else {
		sendNormalResponse(w, string(resp), 200)
	}
}
