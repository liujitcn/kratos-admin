package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestPreferredPluginConnection 优先保留可用连接，避免历史失败记录遮蔽成功连接。
func TestPreferredPluginConnection(t *testing.T) {
	rows := []*models.AiPluginConnection{{ID: 2, State: "error"}, {ID: 1, State: "connected"}}
	preferred := preferredPluginConnection(rows)
	if preferred == nil || preferred.ID != 1 {
		t.Fatalf("期望保留已连接记录，得到 %#v", preferred)
	}
}
