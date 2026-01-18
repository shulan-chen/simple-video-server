package main

import (
	"video-server/api"
	"video-server/api/utils"
	"video-server/stream"
	"video-server/web"
)

func main() {
	utils.InitLogging()
	go api.Start()
	go stream.Start()
	web.Start()
}
