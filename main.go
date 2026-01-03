package main

import (
	"video-server/api/utils"
	"video-server/stream"
)

func main() {
	utils.InitLogging()
	//api.Start()

	stream.Start()
}
