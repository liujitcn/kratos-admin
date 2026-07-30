package task

import (
	"context"
	"strings"
	"testing"
)

type taskExecFunc func(map[string]string) ([]string, error)

// Exec 将测试函数适配为任务执行器。
func (f taskExecFunc) Exec(args map[string]string) ([]string, error) {
	return f(args)
}

// TestSchedulerRunRecovery 验证任务 panic 被转为错误并交给观察器。
func TestSchedulerRunRecovery(t *testing.T) {
	registry := NewRegistry()
	var err error
	err = registry.Register(Task{
		Name: "panic-task",
		Args: map[string]string{"source": "registered"},
		Exec: taskExecFunc(func(map[string]string) ([]string, error) {
			panic("failed")
		}),
	})
	if err != nil {
		t.Fatalf("注册任务失败: %v", err)
	}

	var observed Result
	var scheduler *Scheduler
	scheduler, err = NewScheduler(registry, ObserverFunc(func(result Result) {
		observed = result
	}))
	if err != nil {
		t.Fatalf("创建任务调度器失败: %v", err)
	}
	_, err = scheduler.Run(context.Background(), "panic-task", map[string]string{"source": "manual"})
	if err == nil || !strings.Contains(err.Error(), "任务执行异常") {
		t.Fatalf("任务 panic 未被转换: %v", err)
	}
	if observed.Name != "panic-task" || observed.Err == nil {
		t.Fatalf("任务观察结果错误: %+v", observed)
	}
}
