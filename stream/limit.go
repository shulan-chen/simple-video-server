package stream

import "video-server/api/utils"

type ConnLimiter struct {
	concurrentConn int
	bucket         chan int
}

func NewConnLimiter(cc int) *ConnLimiter {
	return &ConnLimiter{
		concurrentConn: cc,
		bucket:         make(chan int, cc),
	}
}
func (cl *ConnLimiter) GetConn() bool {
	if len(cl.bucket) >= cl.concurrentConn {
		utils.Logger.Error("Reached the rate limit of connections")
		return false
	}
	cl.bucket <- 1
	return true
}

func (cl *ConnLimiter) Release() {
	_ = <-cl.bucket
	//utils.Logger.Info("Release one connection, current connection: %d", c)
}
