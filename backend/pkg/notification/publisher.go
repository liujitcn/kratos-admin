package notification

import (
	"context"
	"fmt"
	"sync/atomic"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// Audience 描述站内信的单项受众规则。
type Audience struct {
	// Type 是受众类型。
	Type basev1.MessageAudienceType
	// ID 是受众编号，租户全员时为 0。
	ID int64
	// IncludeChildren 表示部门受众是否包含子部门。
	IncludeChildren bool
}

// Message 描述业务模块发布站内信所需的消息内容。
type Message struct {
	// TenantID 是消息所属租户编号。
	TenantID int64
	// CategoryCode 是消息分类稳定编码。
	CategoryCode string
	// Title 是消息标题。
	Title string
	// Content 是消息正文。
	Content string
	// ContentFormat 是消息正文格式。
	ContentFormat basev1.MessageContentFormat
	// Priority 是消息优先级。
	Priority basev1.MessagePriority
	// ActionType 是消息动作类型。
	ActionType basev1.MessageActionType
	// ActionTarget 是消息动作目标。
	ActionTarget string
	// ActionParams 是消息动作参数 JSON。
	ActionParams string
	// Audiences 是消息受众规则。
	Audiences []Audience
	// Source 是业务来源标识。
	Source string
	// IdempotencyKey 是租户内来源幂等键。
	IdempotencyKey string
	// SenderName 是发送者显示名称。
	SenderName string
	// ExpiresAt 是消息过期时间戳，单位为毫秒。
	ExpiresAt int64
}

// Publisher 定义业务模块发布站内信的能力。
type Publisher interface {
	Publish(context.Context, Message) (int64, error)
}

// PublisherFunc 将函数适配为 Publisher。
type PublisherFunc func(context.Context, Message) (int64, error)

// Publish 调用适配的发布函数。
func (f PublisherFunc) Publish(ctx context.Context, message Message) (int64, error) {
	return f(ctx, message)
}

// Publish 调用当前进程注册的站内信发布方。
func Publish(ctx context.Context, message Message) (int64, error) {
	value := defaultPublisher.Load()
	if value == nil || value.publisher == nil {
		return 0, fmt.Errorf("站内信发布方未初始化")
	}
	return value.publisher.Publish(ctx, message)
}

// SetDefaultPublisher 设置当前进程默认站内信发布方。
func SetDefaultPublisher(publisher Publisher) {
	defaultPublisher.Store(&publisherHolder{publisher: publisher})
}

// publisherHolder 保存当前进程注册的站内信发布器，供原子指针安全替换。
type publisherHolder struct {
	publisher Publisher // 当前默认发布器实例。
}

var defaultPublisher atomic.Pointer[publisherHolder]
