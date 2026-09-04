package admin

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	logmiddleware "github.com/liujitcn/kratos-admin/backend/internal/server/middleware/log"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// BaseLogFallbackTaskName 是日志入库回退任务的稳定调用目标。
	BaseLogFallbackTaskName  = "system.admin.BaseLogFallback"
	baseLogFallbackBatchSize = 1000
)

var _ cron.TaskExec = (*BaseLogFallbackTask)(nil)

// BaseLogFallbackTask 将日志入库回退文件中的事件重新写入对应日志表。
type BaseLogFallbackTask struct {
	configCache    cache.Cache
	loginRepo      *data.BaseLoginLogRepository
	operationRepo  *data.BaseOperationLogRepository
	dataAccessRepo *data.BaseDataAccessLogRepository
	permissionRepo *data.BasePermissionLogRepository
}

// NewBaseLogFallbackTask 创建日志入库回退任务。
func NewBaseLogFallbackTask(
	baseCase *biz.BaseCase,
	loginRepo *data.BaseLoginLogRepository,
	operationRepo *data.BaseOperationLogRepository,
	dataAccessRepo *data.BaseDataAccessLogRepository,
	permissionRepo *data.BasePermissionLogRepository,
) *BaseLogFallbackTask {
	return &BaseLogFallbackTask{
		configCache:    baseCase.Cache,
		loginRepo:      loginRepo,
		operationRepo:  operationRepo,
		dataAccessRepo: dataAccessRepo,
		permissionRepo: permissionRepo,
	}
}

// Task 返回交由 base_job 调度的任务定义。
func (t *BaseLogFallbackTask) Task() cron.Task {
	return cron.Task{Name: BaseLogFallbackTaskName, Exec: t}
}

// Exec 扫描日志入库回退文件并重新写入待处理事件。
func (t *BaseLogFallbackTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	config := runtimeconfig.DefaultBaseLogFallbackConfig()
	err := runtimeconfig.LoadJSON(t.configCache, runtimeconfig.BaseLogFallbackKey, &config)
	if err != nil {
		return nil, fmt.Errorf("读取日志入库回退配置失败: %w", err)
	}
	var replayed int
	replayed, err = t.replayBaseLogFallback(ctx, config)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("日志入库回退 %d 条", replayed)}, nil
}

type baseLogFallbackRecord struct {
	Content json.RawMessage `json:"content"`
	HMAC    string          `json:"hmac"`
}

type baseLogFallbackContent struct {
	RecordedAt string          `json:"recorded_at"`
	Stage      string          `json:"stage"`
	Kind       string          `json:"kind"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
}

type baseLogFallbackAdminEvent struct {
	EventID string          `json:"event_id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

// replayBaseLogFallback 读取日志入库回退文件并将完整事件重新写入日志表。
func (t *BaseLogFallbackTask) replayBaseLogFallback(ctx context.Context, config runtimeconfig.BaseLogFallbackConfig) (int, error) {
	path := runtimeconfig.BaseLogFallbackFilePath(config.FilePath)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("检查日志入库回退文件失败: %w", err)
	}
	var integrityKey string
	integrityKey, err = runtimeconfig.ResolveBaseLogFallbackIntegrityKey()
	if err != nil {
		return 0, err
	}
	return replayBaseLogFallbackFile(ctx, path, integrityKey, func(ctx context.Context, eventID string, rawValue []byte) error {
		return logmiddleware.ReplayAdminEvent(ctx, eventID, rawValue, t.loginRepo, t.operationRepo, t.dataAccessRepo, t.permissionRepo)
	})
}

// replayBaseLogFallbackFile 按持久化偏移量逐行处理日志入库回退文件。
func replayBaseLogFallbackFile(ctx context.Context, path, integrityKey string, replay func(context.Context, string, []byte) error) (int, error) {
	var file *os.File
	var err error
	file, err = os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("打开日志入库回退文件失败: %w", err)
	}
	defer file.Close()
	var fileInfo os.FileInfo
	fileInfo, err = file.Stat()
	if err != nil {
		return 0, fmt.Errorf("读取日志入库回退文件信息失败: %w", err)
	}
	var offset int64
	offset, err = readBaseLogFallbackOffset(path)
	if err != nil {
		return 0, err
	}
	if offset < 0 || offset > fileInfo.Size() {
		offset = 0
	}
	if offset == fileInfo.Size() {
		return 0, nil
	}
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("定位日志入库回退文件偏移失败: %w", err)
	}
	reader := bufio.NewReader(file)
	currentOffset := offset
	replayed := 0
	for replayed < baseLogFallbackBatchSize {
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.EOF {
			return replayed, fmt.Errorf("读取日志入库回退记录失败: %w", readErr)
		}
		if len(line) == 0 || line[len(line)-1] != '\n' {
			break
		}
		nextOffset := currentOffset + int64(len(line))
		line = bytes.TrimSpace(bytes.TrimSuffix(line, []byte{'\n'}))
		if len(line) == 0 {
			currentOffset = nextOffset
			if err = writeBaseLogFallbackOffset(path, currentOffset); err != nil {
				return replayed, err
			}
			continue
		}
		var eventID string
		var payload []byte
		eventID, payload, err = decodeBaseLogFallbackRecord(line, integrityKey)
		if err != nil {
			return replayed, fmt.Errorf("解析日志入库回退记录失败，偏移 %d: %w", currentOffset, err)
		}
		if err = replay(ctx, eventID, payload); err != nil {
			return replayed, fmt.Errorf("重新写入日志入库回退记录失败，偏移 %d: %w", currentOffset, err)
		}
		currentOffset = nextOffset
		if err = writeBaseLogFallbackOffset(path, currentOffset); err != nil {
			return replayed, err
		}
		replayed++
	}
	return replayed, nil
}

// decodeBaseLogFallbackRecord 校验日志入库回退记录并提取可重新写入的事件载荷。
func decodeBaseLogFallbackRecord(line []byte, integrityKey string) (string, []byte, error) {
	var record baseLogFallbackRecord
	err := json.Unmarshal(line, &record)
	if err != nil {
		return "", nil, err
	}
	if len(record.Content) == 0 || record.HMAC == "" {
		return "", nil, fmt.Errorf("日志入库回退记录缺少内容或 HMAC")
	}
	macValue := hmac.New(sha256.New, []byte(integrityKey))
	_, _ = macValue.Write(record.Content)
	expectedHMAC := hex.EncodeToString(macValue.Sum(nil))
	if !hmac.Equal([]byte(expectedHMAC), []byte(record.HMAC)) {
		return "", nil, fmt.Errorf("日志入库回退记录 HMAC 校验失败")
	}
	var content baseLogFallbackContent
	err = json.Unmarshal(record.Content, &content)
	if err != nil {
		return "", nil, fmt.Errorf("解析日志入库回退内容失败: %w", err)
	}
	var event baseLogFallbackAdminEvent
	err = json.Unmarshal(content.Payload, &event)
	if err != nil {
		return "", nil, fmt.Errorf("解析 Admin 审计事件失败: %w", err)
	}
	if event.Kind == "" || len(event.Payload) == 0 {
		return "", nil, fmt.Errorf("日志入库回退内容不是可重新写入的 Admin 事件")
	}
	eventID := event.EventID
	if eventID == "" {
		digest := sha256.Sum256(record.Content)
		eventID = "base-log-fallback-" + hex.EncodeToString(digest[:])
	}
	return eventID, content.Payload, nil
}

// readBaseLogFallbackOffset 读取日志入库回退文件的已确认字节偏移量。
func readBaseLogFallbackOffset(path string) (int64, error) {
	offsetPath := path + ".offset"
	data, err := os.ReadFile(offsetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("读取日志入库回退偏移量失败: %w", err)
	}
	var offset int64
	offset, err = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || offset < 0 {
		if err == nil {
			err = fmt.Errorf("偏移量不能为负数")
		}
		return 0, fmt.Errorf("日志入库回退偏移量无效: %w", err)
	}
	return offset, nil
}

// writeBaseLogFallbackOffset 原子更新日志入库回退文件的已确认字节偏移量。
func writeBaseLogFallbackOffset(path string, offset int64) error {
	offsetPath := path + ".offset"
	temporaryPath := offsetPath + ".tmp"
	content := []byte(strconv.FormatInt(offset, 10) + "\n")
	err := os.MkdirAll(filepath.Dir(offsetPath), 0o750)
	if err != nil {
		return fmt.Errorf("创建日志入库回退偏移量目录失败: %w", err)
	}
	err = os.WriteFile(temporaryPath, content, 0o600)
	if err != nil {
		return fmt.Errorf("写入日志入库回退偏移量失败: %w", err)
	}
	err = os.Rename(temporaryPath, offsetPath)
	if err != nil {
		return fmt.Errorf("提交日志入库回退偏移量失败: %w", err)
	}
	return nil
}
