// Package startup 管理服务启动前的初始化钩子及其清理工作。
package startup

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Hook 描述一项需要在服务启动前初始化并在退出时清理的工作。
type Hook struct {
	// Name 是钩子的稳定唯一名称。
	Name string
	// Start 在传输服务启动前执行。
	Start func(context.Context) error
	// Stop 在应用清理阶段执行。
	Stop func(context.Context) error
}

// Manager 按注册顺序执行启动钩子，并按相反顺序清理已启动钩子。
type Manager struct {
	lifecycleMu sync.Mutex
	hooks       []Hook
	started     []Hook
	running     bool
}

// NewManager 创建空的启动钩子管理器。
func NewManager() *Manager {
	return &Manager{}
}

// Register 注册启动钩子，并拒绝空名称、空启动方法和重复名称。
func (m *Manager) Register(hooks ...Hook) error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.running {
		return fmt.Errorf("启动钩子管理器已启动，不能追加钩子")
	}
	names := make(map[string]struct{}, len(m.hooks)+len(hooks))
	for _, hook := range m.hooks {
		names[hook.Name] = struct{}{}
	}
	for _, hook := range hooks {
		if hook.Name == "" {
			return fmt.Errorf("启动钩子名称不能为空")
		}
		if hook.Start == nil {
			return fmt.Errorf("启动钩子启动方法不能为空: %s", hook.Name)
		}
		if _, exists := names[hook.Name]; exists {
			return fmt.Errorf("启动钩子名称重复: %s", hook.Name)
		}
		names[hook.Name] = struct{}{}
	}
	m.hooks = append(m.hooks, hooks...)
	return nil
}

// Start 按注册顺序执行钩子，失败时逆序清理已经启动的钩子。
func (m *Manager) Start(ctx context.Context) error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.running {
		return fmt.Errorf("启动钩子管理器已启动")
	}
	m.running = true
	m.started = nil
	var err error
	for _, hook := range m.hooks {
		err = hook.Start(ctx)
		if err != nil {
			stopErr := m.stopStarted(ctx)
			m.running = false
			m.started = nil
			if stopErr != nil {
				return errors.Join(fmt.Errorf("执行启动钩子 %q: %w", hook.Name, err), stopErr)
			}
			return fmt.Errorf("执行启动钩子 %q: %w", hook.Name, err)
		}
		m.started = append(m.started, hook)
	}
	return nil
}

// Stop 按启动顺序的相反方向清理钩子，并汇总全部清理错误。
func (m *Manager) Stop(ctx context.Context) error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if !m.running {
		return nil
	}
	stopErr := m.stopStarted(ctx)
	m.running = false
	m.started = nil
	return stopErr
}

// stopStarted 按启动顺序的相反方向清理已启动钩子。
func (m *Manager) stopStarted(ctx context.Context) error {
	var stopErr error
	for index := len(m.started) - 1; index >= 0; index-- {
		hook := m.started[index]
		if hook.Stop == nil {
			continue
		}
		var err error
		err = hook.Stop(ctx)
		if err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("清理启动钩子 %q: %w", hook.Name, err))
		}
	}
	return stopErr
}
