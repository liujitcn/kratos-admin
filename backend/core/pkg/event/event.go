package event

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// Handler 处理一条进程内事件。
type Handler[T any] func(context.Context, T) error

// Bus 向当前进程中已注册的订阅者发布同一类型的事件。
type Bus[T any] struct {
	mu          sync.RWMutex
	subscribers map[uint64]Handler[T]
	sequence    atomic.Uint64
}

// NewBus 创建指定事件类型的发布订阅总线。
func NewBus[T any]() *Bus[T] {
	return &Bus[T]{subscribers: make(map[uint64]Handler[T])}
}

// Subscribe 注册事件处理器，并返回可重复调用的取消函数。
func (b *Bus[T]) Subscribe(handler Handler[T]) func() {
	if handler == nil {
		return func() {}
	}
	id := b.sequence.Add(1)
	b.mu.Lock()
	b.subscribers[id] = handler
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, id)
			b.mu.Unlock()
		})
	}
}

// Publish 向当前订阅快照发布事件，并汇总全部处理错误。
func (b *Bus[T]) Publish(ctx context.Context, event T) error {
	var err error
	for _, handler := range b.snapshot() {
		err = errors.Join(err, handler(ctx, event))
	}
	return err
}

func (b *Bus[T]) snapshot() []Handler[T] {
	b.mu.RLock()
	defer b.mu.RUnlock()

	handlers := make([]Handler[T], 0, len(b.subscribers))
	for _, handler := range b.subscribers {
		handlers = append(handlers, handler)
	}
	return handlers
}
