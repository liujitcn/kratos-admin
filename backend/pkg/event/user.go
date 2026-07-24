package event

import (
	"sync"
	"sync/atomic"
)

// UserSubscriber 接收基础用户数据变更通知。
type UserSubscriber interface {
	UserChanged(userID int64)
	UsersDeleted(userIDs []int64)
}

// UserEvents 向已装配模块发布基础用户数据变更通知。
type UserEvents struct {
	mu          sync.RWMutex
	subscribers map[uint64]UserSubscriber
	sequence    atomic.Uint64
}

// NewUserEvents 创建不包含业务订阅者的用户变更通知发布器。
func NewUserEvents() *UserEvents {
	return &UserEvents{
		subscribers: make(map[uint64]UserSubscriber),
	}
}

// Subscribe 注册用户变更订阅者，并返回可重复调用的取消函数。
func (e *UserEvents) Subscribe(subscriber UserSubscriber) func() {
	if e == nil || subscriber == nil {
		return func() {}
	}

	id := e.sequence.Add(1)
	e.mu.Lock()
	e.subscribers[id] = subscriber
	e.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			e.mu.Lock()
			delete(e.subscribers, id)
			e.mu.Unlock()
		})
	}
}

// PublishUserChanged 发布单个用户新增、更新或状态变化通知。
func (e *UserEvents) PublishUserChanged(userID int64) {
	if e == nil || userID <= 0 {
		return
	}
	for _, subscriber := range e.snapshot() {
		subscriber.UserChanged(userID)
	}
}

// PublishUsersDeleted 发布批量用户删除通知。
func (e *UserEvents) PublishUsersDeleted(userIDs []int64) {
	if e == nil || len(userIDs) == 0 {
		return
	}
	clonedUserIDs := append([]int64(nil), userIDs...)
	for _, subscriber := range e.snapshot() {
		subscriber.UsersDeleted(clonedUserIDs)
	}
}

// snapshot 返回当前订阅者副本，使发布过程不阻塞订阅变更。
func (e *UserEvents) snapshot() []UserSubscriber {
	e.mu.RLock()
	subscribers := make([]UserSubscriber, 0, len(e.subscribers))
	for _, subscriber := range e.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	e.mu.RUnlock()
	return subscribers
}
