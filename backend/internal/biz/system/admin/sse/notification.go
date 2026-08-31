package sse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/liujitcn/kratos-kit/transport/sse"
)

const notificationStreamID = "base.notification"

var _ sse.SSEStream = (*Notification)(nil)

// Notification 描述当前登录用户隔离的站内信 SSE 流。
type Notification struct{}

// NewNotification 创建站内信 SSE 流。
func NewNotification() *Notification {
	return &Notification{}
}

// ID 返回站内信 SSE 流标识。
func (*Notification) ID() string {
	return notificationStreamID
}

// Resolve 返回按用户隔离的底层传输流标识。

func (*Notification) Resolve(channelID string, userID int64) (string, error) {
	parts := strings.SplitN(channelID, ":", 2)
	if len(parts) == 2 {
		tenantID, err := strconv.ParseInt(parts[0], 10, 64)
		if err == nil && tenantID > 0 {
			return fmt.Sprintf("%s:%d:%d", notificationStreamID, tenantID, userID), nil
		}
	}
	return fmt.Sprintf("%s:%d", notificationStreamID, userID), nil
}

// ResolveTenant 返回按租户和用户双重隔离的底层传输流标识。
func (*Notification) ResolveTenant(_ string, userID, tenantID int64) (string, error) {
	return fmt.Sprintf("%s:%d:%d", notificationStreamID, tenantID, userID), nil
}
