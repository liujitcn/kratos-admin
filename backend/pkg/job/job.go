package job

import (
	"fmt"
	"strings"
	"time"

	systemcommonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/common/v1"

	"github.com/liujitcn/kratos-admin/backend/pkg/queue"

	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
)

// ExecJob 表示一次定时任务执行上下文。
type ExecJob struct {
	JobID        int64             // 任务ID
	Args         map[string]string // 任务参数
	InvokeTarget TaskExec
	Status       systemcommonv1.BaseJobLogStatus
	ErrMsg       string
}

// LogJobFailureWithInput 使用原始入参记录任务失败日志。
func LogJobFailureWithInput(jobID int64, input string, err error) {
	// 没有实际错误时，无需生成失败日志。
	if err == nil {
		return
	}

	baseJobLog := models.BaseJobLog{
		JobID:       jobID,      // 定时任务id
		Input:       input,      // 任务参数
		ExecuteTime: time.Now(), // 执行时间
	}
	baseJobLog.Status = _const.BASE_JOB_LOG_STATUS_FAIL
	baseJobLog.Error = err.Error()
	baseJobLog.ProcessTime = int32(time.Since(baseJobLog.ExecuteTime).Milliseconds())
	queue.AddQueue(_const.JOB_LOG, baseJobLog)
}

// Execute 执行任务并写入任务日志。
func (e *ExecJob) Execute() (err error) {
	baseJobLog := models.BaseJobLog{
		JobID:       e.JobID,                                // 定时任务id
		Input:       _string.ConvertAnyToJsonString(e.Args), // 任务参数
		ExecuteTime: time.Now(),                             // 执行时间
	}
	ret := make([]string, 0)

	defer func() {
		// 任务执行发生 panic 时，统一转成失败日志并返回错误。
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("任务执行异常: %v", panicValue)
		}

		if err != nil {
			e.Status = systemcommonv1.BaseJobLogStatus(_const.BASE_JOB_LOG_STATUS_FAIL)
			e.ErrMsg = err.Error()
		} else {
			e.Status = systemcommonv1.BaseJobLogStatus(_const.BASE_JOB_LOG_STATUS_SUCCESS)
			e.ErrMsg = ""
		}
		baseJobLog.Output = strings.Join(ret, "<br/>")
		baseJobLog.Status = int32(e.Status)
		baseJobLog.Error = e.ErrMsg
		baseJobLog.ProcessTime = int32(time.Since(baseJobLog.ExecuteTime).Milliseconds())
		queue.AddQueue(_const.JOB_LOG, baseJobLog)
	}()

	ret, err = e.InvokeTarget.Exec(e.Args)
	return err
}
