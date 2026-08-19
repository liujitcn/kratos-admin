package logstream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"google.golang.org/protobuf/proto"
)

const (
	// SSEStreamRuntimeConsole 表示管理后台实时控制台 SSE 流。
	SSEStreamRuntimeConsole = "system.admin.runtime-console"
	// SSEEventRuntimeLog 表示单条实时日志事件。
	SSEEventRuntimeLog = "runtime.log"
	// SSEEventRuntimeGap 表示实时日志丢失提示事件。
	SSEEventRuntimeGap = "runtime.gap"

	defaultBacklogLimit = 200
	maxBacklogLimit     = 500
	maxEntryCount       = 2000
	maxEntryBytes       = 8 << 20
	maxEntryLineBytes   = 64 << 10
	publishQueueSize    = 1024
	sessionTTL          = 30 * time.Minute
)

// Publisher 发布一条结构化 SSE 事件。
type Publisher func(ctx context.Context, streamID, eventID string, payload any)

// Session 保存实时控制台会话和创建时的日志快照。
type Session struct {
	ChannelID      string
	ExpiresAt      time.Time
	InstanceID     string
	LatestSequence string
	Entries        []*adminv1.RuntimeLogEntry
}

// Hub 保存当前进程最近日志、实时控制台会话和异步发布队列。
type Hub struct {
	mu            sync.RWMutex
	entries       []*adminv1.RuntimeLogEntry
	entryBytes    int
	nextSequence  uint64
	instanceID    string
	sessions      map[string]runtimeSession
	publisher     Publisher
	publishQueue  chan queuedEntry
	droppedEvents atomic.Uint64
}

type runtimeSession struct {
	ownerID   int64
	expiresAt time.Time
}

type queuedEntry struct {
	entry        *adminv1.RuntimeLogEntry
	droppedCount uint64
}

var defaultHub = newHub()

// DefaultHub 返回当前进程共享的运行日志中心。
func DefaultHub() *Hub {
	return defaultHub
}

// newHub 创建运行日志中心并启动异步事件转发循环。
func newHub() *Hub {
	hub := &Hub{
		entries:      make([]*adminv1.RuntimeLogEntry, 0, maxEntryCount),
		instanceID:   uuid.NewString(),
		sessions:     make(map[string]runtimeSession),
		publishQueue: make(chan queuedEntry, publishQueueSize),
	}
	go hub.relay()
	return hub
}

// Append 保存并异步广播一条运行日志。
func (h *Hub) Append(entry *adminv1.RuntimeLogEntry) {
	if h == nil || entry == nil {
		return
	}
	stored := proto.Clone(entry).(*adminv1.RuntimeLogEntry)
	if stored.GetTimestamp() == "" {
		stored.Timestamp = time.Now().Format("2006-01-02 15:04:05.000")
	}
	stored.Line, stored.IsTruncated = truncateText(stored.GetLine(), maxEntryLineBytes, stored.GetIsTruncated())
	stored.Message, stored.IsTruncated = truncateText(stored.GetMessage(), maxEntryLineBytes, stored.GetIsTruncated())

	h.mu.Lock()
	h.nextSequence++
	stored.Sequence = strconv.FormatUint(h.nextSequence, 10)
	h.entries = append(h.entries, stored)
	h.entryBytes += runtimeEntrySize(stored)
	for len(h.entries) > maxEntryCount || h.entryBytes > maxEntryBytes {
		h.entryBytes -= runtimeEntrySize(h.entries[0])
		h.entries = h.entries[1:]
	}
	h.mu.Unlock()

	queued := queuedEntry{entry: proto.Clone(stored).(*adminv1.RuntimeLogEntry), droppedCount: h.droppedEvents.Swap(0)}
	select {
	case h.publishQueue <- queued:
	default:
		h.droppedEvents.Add(queued.droppedCount + 1)
	}
}

// OpenSession 创建当前用户专属实时控制台会话并返回最近日志。
func (h *Hub) OpenSession(ownerID int64, backlogLimit int) Session {
	if backlogLimit <= 0 {
		backlogLimit = defaultBacklogLimit
	}
	if backlogLimit > maxBacklogLimit {
		backlogLimit = maxBacklogLimit
	}
	now := time.Now()
	expiresAt := now.Add(sessionTTL)
	channelID := uuid.NewString()

	h.mu.Lock()
	h.pruneSessionsLocked(now)
	h.sessions[channelID] = runtimeSession{ownerID: ownerID, expiresAt: expiresAt}
	start := len(h.entries) - backlogLimit
	if start < 0 {
		start = 0
	}
	entries := cloneEntries(h.entries[start:])
	latestSequence := strconv.FormatUint(h.nextSequence, 10)
	instanceID := h.instanceID
	h.mu.Unlock()

	return Session{
		ChannelID:      channelID,
		ExpiresAt:      expiresAt,
		InstanceID:     instanceID,
		LatestSequence: latestSequence,
		Entries:        entries,
	}
}

// IsSessionOwner 判断实时控制台会话是否有效且属于指定用户。
func (h *Hub) IsSessionOwner(channelID string, ownerID int64) bool {
	if h == nil || channelID == "" {
		return false
	}
	now := time.Now()
	h.mu.Lock()
	session, ok := h.sessions[channelID]
	if ok && !session.expiresAt.After(now) {
		delete(h.sessions, channelID)
		ok = false
	}
	isOwner := ok && session.ownerID == ownerID
	h.mu.Unlock()
	return isOwner
}

// SetPublisher 设置运行日志的 SSE 发布能力。
func (h *Hub) SetPublisher(publisher Publisher) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.publisher = publisher
	h.mu.Unlock()
}

// ClearPublisher 清除运行日志 SSE 发布能力和已创建会话。
func (h *Hub) ClearPublisher() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.publisher = nil
	clear(h.sessions)
	h.mu.Unlock()
}

// relay 从非阻塞队列读取日志并发布到全部有效用户频道。
func (h *Hub) relay() {
	for queued := range h.publishQueue {
		publisher, channelIDs := h.publishTargets(time.Now())
		if publisher == nil || len(channelIDs) == 0 {
			continue
		}
		ctx := context.Background()
		if queued.droppedCount > 0 {
			gap := &adminv1.RuntimeLogGap{
				DroppedCount:   int32(min(queued.droppedCount, uint64(^uint32(0)>>1))),
				LatestSequence: queued.entry.GetSequence(),
			}
			for _, channelID := range channelIDs {
				publisher(ctx, runtimeStreamID(channelID), SSEEventRuntimeGap, gap)
			}
		}
		for _, channelID := range channelIDs {
			publisher(ctx, runtimeStreamID(channelID), SSEEventRuntimeLog, queued.entry)
		}
	}
}

// publishTargets 返回当前发布器和仍在有效期内的频道快照。
func (h *Hub) publishTargets(now time.Time) (Publisher, []string) {
	h.mu.Lock()
	h.pruneSessionsLocked(now)
	publisher := h.publisher
	channelIDs := make([]string, 0, len(h.sessions))
	for channelID := range h.sessions {
		channelIDs = append(channelIDs, channelID)
	}
	h.mu.Unlock()
	return publisher, channelIDs
}

// pruneSessionsLocked 删除全部过期实时控制台会话。
func (h *Hub) pruneSessionsLocked(now time.Time) {
	for channelID, session := range h.sessions {
		if !session.expiresAt.After(now) {
			delete(h.sessions, channelID)
		}
	}
}

// runtimeStreamID 返回指定频道的底层 SSE 流标识。
func runtimeStreamID(channelID string) string {
	return fmt.Sprintf("%s:%s", SSEStreamRuntimeConsole, channelID)
}

// cloneEntries 复制日志条目，避免调用方修改 Hub 内部快照。
func cloneEntries(entries []*adminv1.RuntimeLogEntry) []*adminv1.RuntimeLogEntry {
	cloned := make([]*adminv1.RuntimeLogEntry, 0, len(entries))
	for _, entry := range entries {
		cloned = append(cloned, proto.Clone(entry).(*adminv1.RuntimeLogEntry))
	}
	return cloned
}

// runtimeEntrySize 返回单条日志占用的近似字节数。
func runtimeEntrySize(entry *adminv1.RuntimeLogEntry) int {
	if entry == nil {
		return 0
	}
	return len(entry.GetSequence()) + len(entry.GetTimestamp()) + len(entry.GetLevel()) + len(entry.GetSource()) + len(entry.GetMessage()) + len(entry.GetLine())
}

// truncateText 按 UTF-8 边界截断过长日志文本。
func truncateText(value string, limit int, alreadyTruncated bool) (string, bool) {
	if len(value) <= limit {
		return value, alreadyTruncated
	}
	end := limit
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end] + " ... [truncated]", true
}
