package _const

// Queue 定义项目内统一的队列名称类型。
type Queue string

const (
	// LOG 表示通用日志异步写入队列。
	LOG Queue = "log_queue"
	// JOB_LOG 表示定时任务执行日志队列。
	JOB_LOG Queue = "job_log_queue"
	// TRANSLATION 表示动态资源机器翻译队列。
	TRANSLATION Queue = "translation_queue"
)
