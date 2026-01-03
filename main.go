package main

import (
	api "video-server/api"
	"video-server/api/utils"
)

func main() {
	utils.InitLogging()
	api.Start()
}
