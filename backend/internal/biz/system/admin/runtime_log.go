package biz

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/sse"
)

const (
	defaultRuntimeLogPageLimit = 200
	maxRuntimeLogLineBytes     = 64 << 10
	maxRuntimeLogPageBytes     = 512 << 10
)

// RuntimeLogCase 提供当前进程实时日志和历史日志文件访问能力。
type RuntimeLogCase struct {
	*biz.BaseCase
	hub        *logstream.Hub
	logRoot    string
	signingKey []byte
}

// NewRuntimeLogCase 创建运行日志业务实例并接入 SSE 发布能力。
func NewRuntimeLogCase(baseCase *biz.BaseCase, hub *logstream.Hub, sseRuntime *sse.SSE) (*RuntimeLogCase, error) {
	loggerConfig := baseCase.GetConfig().GetLogger()
	logRoot := ""
	var err error
	if loggerConfig != nil {
		filepathValue := ""
		if loggerConfig.GetZap() != nil {
			filepathValue = loggerConfig.GetZap().GetFilepath()
		} else if loggerConfig.GetZerolog() != nil && strings.EqualFold(loggerConfig.GetZerolog().GetWriter(), "file") {
			filepathValue = loggerConfig.GetZerolog().GetFilepath()
		}
		if filepathValue != "" {
			logRoot, err = filepath.Abs(filepathValue)
			if err != nil {
				return nil, fmt.Errorf("解析运行日志目录失败: %w", err)
			}
		}
	}
	signingKey := make([]byte, 32)
	if _, err = rand.Read(signingKey); err != nil {
		return nil, fmt.Errorf("生成运行日志签名密钥失败: %w", err)
	}
	if hub != nil && sseRuntime != nil {
		hub.SetPublisher(sseRuntime.PublishJSON)
	}
	if logRoot != "" {
		logRoot = filepath.Clean(logRoot)
	}
	return &RuntimeLogCase{
		BaseCase:   baseCase,
		hub:        hub,
		logRoot:    logRoot,
		signingKey: signingKey,
	}, nil
}

// ListRuntimeLogFiles 查询日志目录第一层的历史日志文件。
func (c *RuntimeLogCase) ListRuntimeLogFiles() (*adminv1.ListRuntimeLogFilesResponse, error) {
	if c.logRoot == "" {
		return &adminv1.ListRuntimeLogFilesResponse{Files: []*adminv1.RuntimeLogFile{}}, nil
	}
	entries, err := os.ReadDir(c.logRoot)
	if errors.Is(err, os.ErrNotExist) {
		return &adminv1.ListRuntimeLogFilesResponse{Files: []*adminv1.RuntimeLogFile{}}, nil
	}
	if err != nil {
		return nil, errorsx.Internal("读取日志目录失败").WithCause(err)
	}
	files := make([]*adminv1.RuntimeLogFile, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !isRuntimeLogName(entry.Name()) {
			continue
		}
		var info os.FileInfo
		info, err = entry.Info()
		if err != nil {
			return nil, errorsx.Internal("读取日志文件信息失败").WithCause(err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		files = append(files, &adminv1.RuntimeLogFile{
			FileId:       c.signToken([]byte(entry.Name())),
			Name:         entry.Name(),
			SizeBytes:    info.Size(),
			ModifiedAt:   info.ModTime().Format(time.RFC3339Nano),
			IsCompressed: strings.HasSuffix(strings.ToLower(entry.Name()), ".gz"),
			IsActive:     strings.EqualFold(entry.Name(), "info.log"),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].GetIsActive() != files[j].GetIsActive() {
			return files[i].GetIsActive()
		}
		if files[i].GetModifiedAt() != files[j].GetModifiedAt() {
			return files[i].GetModifiedAt() > files[j].GetModifiedAt()
		}
		return files[i].GetName() < files[j].GetName()
	})
	return &adminv1.ListRuntimeLogFilesResponse{Files: files}, nil
}

// ReadRuntimeLogFile 分页读取指定历史日志文件。
func (c *RuntimeLogCase) ReadRuntimeLogFile(req *adminv1.ReadRuntimeLogFileRequest) (*adminv1.ReadRuntimeLogFileResponse, error) {
	path, info, err := c.resolveRuntimeLogFile(req.GetFileId())
	if err != nil {
		return nil, err
	}
	levels, err := runtimeLogLevelSet(req.GetLevels())
	if err != nil {
		return nil, err
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultRuntimeLogPageLimit
	}
	identity := runtimeLogFileIdentity(info)
	filterHash := runtimeLogFilterHash(req.GetKeyword(), levels)
	targetEnd := int64(-1)
	if req.GetCursor() != "" {
		var cursor dto.RuntimeLogCursor
		if err = c.decodeCursor(req.GetCursor(), &cursor); err != nil || cursor.FileID != req.GetFileId() || cursor.FilterHash != filterHash {
			return nil, errorsx.InvalidArgument("日志翻页游标无效").WithCause(err)
		}
		if cursor.Identity != identity || info.Size() < cursor.Size {
			return &adminv1.ReadRuntimeLogFileResponse{Entries: []*adminv1.RuntimeLogEntry{}, FileChanged: true}, nil
		}
		targetEnd = cursor.End
	}
	entries, matchedEnd, err := scanRuntimeLogFile(path, targetEnd, limit, req.GetKeyword(), levels)
	if err != nil {
		return nil, err
	}
	var currentInfo os.FileInfo
	currentInfo, err = os.Lstat(path)
	if err != nil || !currentInfo.Mode().IsRegular() || runtimeLogFileIdentity(currentInfo) != identity || currentInfo.Size() < info.Size() {
		return &adminv1.ReadRuntimeLogFileResponse{Entries: []*adminv1.RuntimeLogEntry{}, FileChanged: true}, nil
	}
	start := matchedEnd - int64(len(entries))
	response := &adminv1.ReadRuntimeLogFileResponse{
		Entries: entries,
		HasMore: start > 0,
	}
	if response.GetHasMore() {
		response.NextCursor, err = c.encodeCursor(dto.RuntimeLogCursor{
			FileID:     req.GetFileId(),
			Identity:   identity,
			FilterHash: filterHash,
			Size:       info.Size(),
			End:        start,
		})
		if err != nil {
			return nil, errorsx.Internal("生成日志翻页游标失败").WithCause(err)
		}
	}
	return response, nil
}

// OpenRuntimeConsole 创建当前用户专属实时控制台频道。
func (c *RuntimeLogCase) OpenRuntimeConsole(ctx context.Context, req *adminv1.OpenRuntimeConsoleRequest) (*adminv1.OpenRuntimeConsoleResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if c.hub == nil {
		return nil, errorsx.Internal("实时日志服务未初始化")
	}
	session := c.hub.OpenSession(authInfo.UserId, int(req.GetBacklogLimit()))
	return &adminv1.OpenRuntimeConsoleResponse{
		Stream:         logstream.SSEStreamRuntimeConsole,
		ChannelId:      session.ChannelID,
		ExpiresAt:      session.ExpiresAt.Format(time.RFC3339Nano),
		InstanceId:     session.InstanceID,
		LatestSequence: session.LatestSequence,
		Entries:        session.Entries,
	}, nil
}

// DownloadRuntimeLogFile 读取历史日志原文件用于直接下载。
func (c *RuntimeLogCase) DownloadRuntimeLogFile(fileID string) (*dto.RuntimeLogDownload, error) {
	path, _, err := c.resolveRuntimeLogFile(fileID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errorsx.Internal("读取日志下载文件失败").WithCause(err)
	}
	name := filepath.Base(path)
	contentType := "text/plain; charset=utf-8"
	if strings.HasSuffix(strings.ToLower(name), ".gz") {
		contentType = "application/gzip"
	}
	return &dto.RuntimeLogDownload{Name: name, ContentType: contentType, Data: data}, nil
}

// resolveRuntimeLogFile 校验签名文件标识并解析安全的第一层普通文件。
func (c *RuntimeLogCase) resolveRuntimeLogFile(fileID string) (string, os.FileInfo, error) {
	if c.logRoot == "" {
		return "", nil, errorsx.ResourceNotFound("未配置历史日志目录")
	}
	nameBytes, valid := c.verifyToken(fileID)
	name := string(nameBytes)
	if !valid || !isRuntimeLogName(name) || filepath.Base(name) != name {
		return "", nil, errorsx.InvalidArgument("日志文件标识无效")
	}
	path := filepath.Join(c.logRoot, name)
	if filepath.Dir(path) != c.logRoot {
		return "", nil, errorsx.PermissionDenied("禁止访问日志目录外文件")
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil, errorsx.ResourceNotFound("日志文件不存在").WithCause(err)
	}
	if err != nil {
		return "", nil, errorsx.Internal("读取日志文件信息失败").WithCause(err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", nil, errorsx.PermissionDenied("日志文件类型不允许访问")
	}
	return path, info, nil
}

// encodeCursor 签名并编码历史日志翻页游标。
func (c *RuntimeLogCase) encodeCursor(cursor dto.RuntimeLogCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return c.signToken(payload), nil
}

// decodeCursor 校验并解析历史日志翻页游标。
func (c *RuntimeLogCase) decodeCursor(value string, cursor *dto.RuntimeLogCursor) error {
	payload, valid := c.verifyToken(value)
	if !valid {
		return fmt.Errorf("游标签名无效")
	}
	if err := json.Unmarshal(payload, cursor); err != nil {
		return err
	}
	if cursor.End < 0 {
		return fmt.Errorf("游标位置无效")
	}
	return nil
}

// signToken 对文件名或游标负载签名并生成不透明标识。
func (c *RuntimeLogCase) signToken(payload []byte) string {
	mac := hmac.New(sha256.New, c.signingKey)
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// verifyToken 校验不透明标识签名并返回原始负载。
func (c *RuntimeLogCase) verifyToken(value string) ([]byte, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	mac := hmac.New(sha256.New, c.signingKey)
	_, _ = mac.Write(payload)
	return payload, hmac.Equal(signature, mac.Sum(nil))
}

// scanRuntimeLogFile 扫描日志并保留目标位置之前满足数量和字节限制的最近条目。
func scanRuntimeLogFile(path string, targetEnd int64, limit int, keyword string, levels map[string]struct{}) (entries []*adminv1.RuntimeLogEntry, matchedEnd int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, errorsx.Internal("打开日志文件失败").WithCause(err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = errorsx.Internal("关闭日志文件失败").WithCause(closeErr)
		}
	}()
	var reader io.Reader = file
	var gzipReader *gzip.Reader
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gzipReader, err = gzip.NewReader(file)
		if err != nil {
			return nil, 0, errorsx.InvalidArgument("日志压缩文件损坏").WithCause(err)
		}
		defer func() {
			closeErr := gzipReader.Close()
			if err == nil && closeErr != nil {
				err = errorsx.Internal("关闭日志压缩流失败").WithCause(closeErr)
			}
		}()
		reader = gzipReader
	}
	buffered := bufio.NewReaderSize(reader, maxRuntimeLogLineBytes)
	pageBytes := 0
	entries = make([]*adminv1.RuntimeLogEntry, 0, limit)
	for {
		var line string
		var truncated bool
		line, truncated, err = readRuntimeLogLine(buffered)
		if errors.Is(err, io.EOF) {
			err = nil
			break
		}
		if err != nil {
			return nil, 0, errorsx.Internal("读取日志文件失败").WithCause(err)
		}
		entry := logstream.ParseLine(line, truncated)
		if !matchesRuntimeLog(entry, keyword, levels) {
			continue
		}
		if targetEnd >= 0 && matchedEnd >= targetEnd {
			break
		}
		matchedEnd++
		entries = append(entries, entry)
		pageBytes += len(entry.GetLine())
		for len(entries) > limit || pageBytes > maxRuntimeLogPageBytes {
			pageBytes -= len(entries[0].GetLine())
			entries = entries[1:]
		}
	}
	return entries, matchedEnd, nil
}

// readRuntimeLogLine 读取单行日志并丢弃超过单行上限的剩余内容。
func readRuntimeLogLine(reader *bufio.Reader) (string, bool, error) {
	var builder strings.Builder
	truncated := false
	readAny := false
	for {
		fragment, isPrefix, err := reader.ReadLine()
		if len(fragment) > 0 {
			readAny = true
			remaining := maxRuntimeLogLineBytes - builder.Len()
			if remaining > 0 {
				builder.Write(fragment[:min(len(fragment), remaining)])
			}
			if len(fragment) > remaining {
				truncated = true
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) && readAny {
				return runtimeLogLineText(builder.String(), truncated), truncated, nil
			}
			return "", truncated, err
		}
		if !isPrefix {
			return runtimeLogLineText(builder.String(), truncated), truncated, nil
		}
		if builder.Len() >= maxRuntimeLogLineBytes {
			truncated = true
		}
	}
}

// matchesRuntimeLog 判断日志是否满足关键字和级别筛选。
func matchesRuntimeLog(entry *adminv1.RuntimeLogEntry, keyword string, levels map[string]struct{}) bool {
	if len(levels) > 0 {
		if _, ok := levels[entry.GetLevel()]; !ok {
			return false
		}
	}
	return keyword == "" || strings.Contains(strings.ToLower(entry.GetLine()), strings.ToLower(keyword))
}

// runtimeLogLevelSet 校验并规范化日志级别筛选。
func runtimeLogLevelSet(levels []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(levels))
	for _, level := range levels {
		normalized := logstream.NormalizeLevel(level)
		if !logstream.IsSupportedLevel(normalized) {
			return nil, errorsx.InvalidArgument("日志级别筛选无效")
		}
		result[normalized] = struct{}{}
	}
	return result, nil
}

// runtimeLogFilterHash 计算游标绑定的筛选条件摘要。
func runtimeLogFilterHash(keyword string, levels map[string]struct{}) string {
	levelList := make([]string, 0, len(levels))
	for level := range levels {
		levelList = append(levelList, level)
	}
	sort.Strings(levelList)
	digest := sha256.Sum256([]byte(strings.ToLower(keyword) + "\x00" + strings.Join(levelList, ",")))
	return hex.EncodeToString(digest[:])
}

// runtimeLogFileIdentity 返回用于识别日志滚动的底层文件身份。
func runtimeLogFileIdentity(info os.FileInfo) string {
	statValue := reflect.ValueOf(info.Sys())
	if statValue.IsValid() && statValue.Kind() == reflect.Pointer {
		statValue = statValue.Elem()
	}
	if statValue.IsValid() && statValue.Kind() == reflect.Struct {
		device := statValue.FieldByName("Dev")
		inode := statValue.FieldByName("Ino")
		if device.IsValid() && inode.IsValid() && device.CanInterface() && inode.CanInterface() {
			return fmt.Sprintf("%v:%v", device.Interface(), inode.Interface())
		}
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}

// runtimeLogLineText 为被截断的日志行增加明确后缀。
func runtimeLogLineText(line string, truncated bool) string {
	if !truncated {
		return line
	}
	return line + " ... [truncated]"
}

// isRuntimeLogName 判断文件名是否属于允许访问的日志类型。
func isRuntimeLogName(name string) bool {
	lowerName := strings.ToLower(name)
	return strings.HasSuffix(lowerName, ".log") || strings.HasSuffix(lowerName, ".log.gz")
}
