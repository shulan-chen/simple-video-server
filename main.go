package main

import (
	"video-server/api"
	"video-server/api/utils"
	"video-server/scheduler"
	"video-server/stream"
	"video-server/web"
)

func main() {
	utils.InitLogging()
	go api.Start()
	go stream.Start()
	go scheduler.Start()
	web.Start()
}
