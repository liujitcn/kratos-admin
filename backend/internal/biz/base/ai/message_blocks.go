package ai

import (
	"encoding/json"

	"github.com/go-kratos/kratos/v3/log"
)

// AppendReplyBlocks 在已有业务引导块后追加服务端生成的结构化消息块。
func AppendReplyBlocks(current string, appended ...any) string {
	var existing []json.RawMessage
	if current != "" {
		if err := json.Unmarshal([]byte(current), &existing); err != nil {
			log.Warn("旧 AI 消息块无法解析", "error", err)
			existing = nil
		}
	}
	blocks := make([]any, 0, len(existing)+len(appended))
	for _, block := range existing {
		blocks = append(blocks, block)
	}
	raw, err := json.Marshal(append(blocks, appended...))
	if err != nil {
		log.Error("AI 消息块无法序列化", "error", err)
		return current
	}
	return string(raw)
}
