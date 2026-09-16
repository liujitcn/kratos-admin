package projectaccess

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// ProjectKey 唯一标识所属租户下的公共项目。
type ProjectKey struct {
	TenantID  int64
	ProjectID int64
}

// ChangeGuard 由业务模块持有业务锁并完成校验，再执行公共项目变更及其事务提交。
type ChangeGuard func(context.Context, []ProjectKey, func(context.Context) error) error

// Lifecycle 收集宿主业务模块对项目停用和删除的约束。
type Lifecycle struct {
	mu     sync.RWMutex
	guards map[string]ChangeGuard
}

// NewLifecycle 创建宿主共享的项目生命周期注册器。
func NewLifecycle() *Lifecycle { return &Lifecycle{guards: make(map[string]ChangeGuard)} }

// Register 在服务启动前注册具名业务校验，避免同一模块重复注册。
func (l *Lifecycle) Register(name string, guard ChangeGuard) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if name == "" || guard == nil {
		return fmt.Errorf("项目生命周期模块名称和校验不能为空")
	}
	if _, exists := l.guards[name]; exists {
		return fmt.Errorf("项目生命周期模块重复: %s", name)
	}
	l.guards[name] = guard
	return nil
}

// Change 按稳定顺序持有各模块的业务锁，直到公共项目变更事务完成。
func (l *Lifecycle) Change(ctx context.Context, keys []ProjectKey, apply func(context.Context) error) error {
	keys = slices.Clone(keys)
	slices.SortFunc(keys, func(a, b ProjectKey) int {
		if a.TenantID < b.TenantID {
			return -1
		}
		if a.TenantID > b.TenantID {
			return 1
		}
		if a.ProjectID < b.ProjectID {
			return -1
		}
		if a.ProjectID > b.ProjectID {
			return 1
		}
		return 0
	})
	keys = slices.Compact(keys)
	l.mu.RLock()
	names := make([]string, 0, len(l.guards))
	for name := range l.guards {
		names = append(names, name)
	}
	slices.Sort(names)
	guards := make([]ChangeGuard, 0, len(names))
	for _, name := range names {
		guards = append(guards, l.guards[name])
	}
	l.mu.RUnlock()
	next := apply
	for index := len(guards) - 1; index >= 0; index-- {
		guard := guards[index]
		inner := next
		next = func(ctx context.Context) error { return guard(ctx, keys, inner) }
	}
	return next(ctx)
}
