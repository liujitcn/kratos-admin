package job

import (
	"fmt"
	"sync"
)

// TaskExec 定义定时任务执行器接口。
type TaskExec interface {
	Exec(arg map[string]string) ([]string, error)
}

// Task 表示模块向调度运行时贡献的具名任务。
type Task struct {
	Name string
	Exec TaskExec
}

// Registry 保存已装配模块贡献的定时任务执行器。
type Registry struct {
	mu    sync.RWMutex
	tasks map[string]TaskExec
}

// NewRegistry 创建空的定时任务注册表。
func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]TaskExec),
	}
}

// Register 注册一组具名任务，并拒绝重复或不完整的任务贡献。
func (r *Registry) Register(tasks ...Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registered := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		if task.Name == "" {
			return fmt.Errorf("定时任务名称不能为空")
		}
		if task.Exec == nil {
			return fmt.Errorf("定时任务执行器不能为空: %s", task.Name)
		}
		if _, exists := r.tasks[task.Name]; exists {
			return fmt.Errorf("定时任务名称重复: %s", task.Name)
		}
		if _, exists := registered[task.Name]; exists {
			return fmt.Errorf("定时任务名称重复: %s", task.Name)
		}
		registered[task.Name] = struct{}{}
	}
	for _, task := range tasks {
		r.tasks[task.Name] = task.Exec
	}
	return nil
}

// Lookup 按名称查询已注册的任务执行器。
func (r *Registry) Lookup(name string) (TaskExec, bool) {
	r.mu.RLock()
	exec, exists := r.tasks[name]
	r.mu.RUnlock()
	return exec, exists
}
