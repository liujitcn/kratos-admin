// Package task 提供模块无关的任务注册、执行和 Cron 调度能力。
package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
)

// TaskExec 定义兼容既有任务实现的执行接口。
type TaskExec interface {
	Exec(arg map[string]string) ([]string, error)
}

// ContextTaskExec 定义可以接收应用上下文的任务执行接口。
type ContextTaskExec interface {
	ExecContext(context.Context, map[string]string) ([]string, error)
}

// Task 表示模块向调度运行时贡献的具名任务。
type Task struct {
	// Name 是任务的稳定唯一名称。
	Name string
	// Expression 是可选的 Cron 表达式，空值表示仅注册、不自动调度。
	Expression string
	// Args 是自动调度时传递给执行器的参数。
	Args map[string]string
	// Exec 是任务执行器。
	Exec TaskExec
}

// Result 描述一次任务执行结果。
type Result struct {
	// Name 是任务名称。
	Name string
	// StartedAt 是任务开始时间。
	StartedAt time.Time
	// Duration 是任务执行耗时。
	Duration time.Duration
	// Output 是任务输出。
	Output []string
	// Err 是任务执行错误。
	Err error
}

// Observer 接收任务执行结果，业务日志和指标通过适配器实现。
type Observer interface {
	Observe(Result)
}

// ObserverFunc 将函数适配为任务执行观察器。
type ObserverFunc func(Result)

// Observe 接收任务执行结果。
func (f ObserverFunc) Observe(result Result) {
	if f != nil {
		f(result)
	}
}

// Registry 保存已装配模块贡献的任务执行器。
type Registry struct {
	mu    sync.RWMutex
	tasks map[string]Task
	order []string
}

// NewRegistry 创建空的任务注册表。
func NewRegistry() *Registry {
	return &Registry{tasks: make(map[string]Task)}
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
		task.Args = cloneArgs(task.Args)
		r.tasks[task.Name] = task
		r.order = append(r.order, task.Name)
	}
	return nil
}

// Lookup 按名称查询已注册的任务执行器。
func (r *Registry) Lookup(name string) (TaskExec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[name]
	return task.Exec, exists
}

// Tasks 返回按注册顺序排列的任务快照。
func (r *Registry) Tasks() []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]Task, 0, len(r.order))
	for _, name := range r.order {
		task := r.tasks[name]
		task.Args = cloneArgs(task.Args)
		tasks = append(tasks, task)
	}
	return tasks
}

// Scheduled 判断注册表是否包含自动调度任务。
func (r *Registry) Scheduled() bool {
	for _, task := range r.Tasks() {
		if task.Expression != "" {
			return true
		}
	}
	return false
}

// Scheduler 将具备 Cron 表达式的任务接入 Kratos 服务生命周期。
type Scheduler struct {
	server   *cronTransport.Server
	registry *Registry
	observer Observer
}

// NewScheduler 创建任务调度服务，并在启动前注册全部静态调度项。
func NewScheduler(registry *Registry, observer Observer) (*Scheduler, error) {
	if registry == nil {
		return nil, fmt.Errorf("任务注册表不能为空")
	}
	scheduler := &Scheduler{
		server:   cronTransport.NewServer(),
		registry: registry,
		observer: observer,
	}
	var err error
	for _, task := range registry.Tasks() {
		if task.Expression == "" {
			continue
		}
		registeredTask := task
		_, err = scheduler.server.NewTimerJob(task.Expression, func() {
			var runErr error
			_, runErr = scheduler.Run(context.Background(), registeredTask.Name, registeredTask.Args)
			if runErr != nil {
				log.Error("定时任务执行失败", "name", registeredTask.Name, "error", runErr)
			}
		})
		if err != nil {
			return nil, fmt.Errorf("注册定时任务 %q: %w", task.Name, err)
		}
	}
	return scheduler, nil
}

// Start 启动 Cron 调度服务。
func (s *Scheduler) Start(ctx context.Context) error {
	return s.server.Start(ctx)
}

// Stop 停止 Cron 调度服务并等待正在执行的任务结束。
func (s *Scheduler) Stop(ctx context.Context) error {
	return s.server.Stop(ctx)
}

// Run 立即执行指定任务，并将结果交给观察器。
func (s *Scheduler) Run(ctx context.Context, name string, args map[string]string) (output []string, err error) {
	exec, exists := s.registry.Lookup(name)
	if !exists || exec == nil {
		return nil, fmt.Errorf("定时任务不存在: %s", name)
	}

	startedAt := time.Now()
	defer func() {
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("任务执行异常: %v", panicValue)
		}
		if s.observer != nil {
			s.observer.Observe(Result{
				Name:      name,
				StartedAt: startedAt,
				Duration:  time.Since(startedAt),
				Output:    append([]string(nil), output...),
				Err:       err,
			})
		}
	}()

	if contextExec, ok := exec.(ContextTaskExec); ok {
		return contextExec.ExecContext(ctx, cloneArgs(args))
	}
	return exec.Exec(cloneArgs(args))
}

func cloneArgs(args map[string]string) map[string]string {
	cloned := make(map[string]string, len(args))
	for key, value := range args {
		cloned[key] = value
	}
	return cloned
}
