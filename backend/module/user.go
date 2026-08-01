package module

// UserSubscriber 接收基础用户数据变更通知。
type UserSubscriber interface {
	// UserChanged 接收单个用户新增、更新或状态变化通知。
	UserChanged(userID int64)
	// UsersDeleted 接收批量用户删除通知。
	UsersDeleted(userIDs []int64)
}

// UserSubscriberContributor 表示可订阅基础用户变更的扩展模块。
type UserSubscriberContributor interface {
	// UserSubscribers 返回模块提供的用户变更订阅者。
	UserSubscribers() []UserSubscriber
}
