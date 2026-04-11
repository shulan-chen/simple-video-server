package scheduler

// 控制信号常量
const (
	READY_TO_DISPATCH = "d" // 准备分发任务
	READY_TO_EXECUTE  = "e" // 准备执行任务
	CLOSE             = "c" // 关闭信号
)

// controlChannel 控制信号通道，用于协调 Dispatcher 和 Executor 的执行顺序
type controlChannel chan string

// dataChannel 数据通道，用于传递任务数据（从 Dispatcher 到 Executor）
type dataChannel chan any

// fn 任务处理函数类型
// Dispatcher 函数：从数据源读取任务并推送到 dataChannel
// Executor 函数：从 dataChannel 读取任务并执行
type fn func(dc dataChannel) error
