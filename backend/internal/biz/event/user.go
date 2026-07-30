package event

import (
	"context"

	coreEvent "github.com/liujitcn/kratos-admin/backend/core/pkg/event"
)

type userEventType uint8

const (
	userChanged userEventType = iota + 1
	usersDeleted
)

type userEvent struct {
	typeID  userEventType
	userID  int64
	userIDs []int64
}

// UserSubscriber 接收基础用户数据变更通知。
type UserSubscriber interface {
	UserChanged(userID int64)
	UsersDeleted(userIDs []int64)
}

// UserEvents 向已装配模块发布基础用户数据变更通知。
type UserEvents struct {
	bus *coreEvent.Bus[userEvent]
}

// NewUserEvents 创建不包含业务订阅者的用户变更通知发布器。
func NewUserEvents() *UserEvents {
	return &UserEvents{bus: coreEvent.NewBus[userEvent]()}
}

// Subscribe 注册用户变更订阅者，并返回可重复调用的取消函数。
func (e *UserEvents) Subscribe(subscriber UserSubscriber) func() {
	if e == nil || subscriber == nil {
		return func() {}
	}
	return e.bus.Subscribe(func(_ context.Context, event userEvent) error {
		switch event.typeID {
		case userChanged:
			subscriber.UserChanged(event.userID)
		case usersDeleted:
			subscriber.UsersDeleted(append([]int64(nil), event.userIDs...))
		}
		return nil
	})
}

// PublishUserChanged 发布单个用户新增、更新或状态变化通知。
func (e *UserEvents) PublishUserChanged(userID int64) {
	if e == nil || userID <= 0 {
		return
	}
	_ = e.bus.Publish(context.Background(), userEvent{
		typeID: userChanged,
		userID: userID,
	})
}

// PublishUsersDeleted 发布批量用户删除通知。
func (e *UserEvents) PublishUsersDeleted(userIDs []int64) {
	if e == nil || len(userIDs) == 0 {
		return
	}
	_ = e.bus.Publish(context.Background(), userEvent{
		typeID:  usersDeleted,
		userIDs: append([]int64(nil), userIDs...),
	})
}
