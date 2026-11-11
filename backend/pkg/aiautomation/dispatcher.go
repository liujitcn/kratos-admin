package aiautomation

import (
	"context"
	"fmt"
	"sync"
)

// Dispatcher 在宿主任务与模块间共享已装配的 AI 执行链，避免重复构造运行时。
type Dispatcher struct {
	ctx     context.Context
	mutex   sync.RWMutex
	handler func(context.Context) error
}

// NewDispatcher 创建与宿主生命周期绑定的调度入口。
func NewDispatcher() (*Dispatcher, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{ctx: ctx}, cancel
}

// Bind 在模块完成装配后登记唯一执行链。
func (d *Dispatcher) Bind(handler func(context.Context) error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.handler = handler
}

// Dispatch 由单一内部任务唤醒，后台执行不依赖浏览器连接。
func (d *Dispatcher) Dispatch(context.Context) error {
	d.mutex.RLock()
	handler := d.handler
	d.mutex.RUnlock()
	if handler == nil {
		return fmt.Errorf("AI自动化模块尚未装配")
	}
	return handler(d.ctx)
}
